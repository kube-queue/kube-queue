package communicate

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"

	queue "github.com/kube-queue/kube-queue/pkg/apis/queue/v1alpha1"
	"github.com/kube-queue/kube-queue/pkg/comm/utils"
	"github.com/kube-queue/kube-queue/pkg/controller"
	"golang.org/x/net/context"
)

type QueueServer struct {
	queueController *controller.Controller
}

func MakeQueueServer(qc *controller.Controller) QueueServerInterface {
	return &QueueServer{queueController: qc}
}

func (cs *QueueServer) AddOrUpdate(ctx context.Context, in *queue.QueueUnit) error {
	cs.queueController.EnqueueItem(in)
	return nil
}

func (cs *QueueServer) Delete(ctx context.Context, in *queue.QueueUnit) error {
	return cs.queueController.DequeueItem(in.Namespace, in.Name, string(in.UID), in.Spec.JobType)
}

func StartServer(csi QueueServerInterface, unParsedAddr string) error {
	protocol, addr, err := utils.ParseEndpoint(unParsedAddr)
	if err != nil {
		return err
	}

	lis, err := net.Listen(protocol, addr)
	if err != nil {
		return err
	}

	http.HandleFunc("/",
		func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/" {
				http.Error(
					w,
					fmt.Sprintf("Path %s not found", r.URL.Path),
					http.StatusNotFound)
				return
			}

			data, err := ioutil.ReadAll(r.Body)
			defer r.Body.Close()
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			var qu queue.QueueUnit
			err = json.Unmarshal(data, &qu)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			switch r.Method {
			case http.MethodPost:
				err = csi.AddOrUpdate(context.Background(), &qu)
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
			case http.MethodDelete:
				err = csi.Delete(context.Background(), &qu)
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
			default:
				http.Error(
					w,
					fmt.Sprintf("method %s not found", r.URL.Path),
					http.StatusNotFound)
				return
			}
		})

	server := &http.Server{
		Handler: nil,
	}

	return server.Serve(lis)
}
