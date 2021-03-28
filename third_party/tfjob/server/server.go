package server

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"

	queue "github.com/kube-queue/kube-queue/pkg/apis/queue/v1alpha1"
	communicate "github.com/kube-queue/kube-queue/pkg/comm/extension"
	"github.com/kube-queue/kube-queue/pkg/comm/utils"
	tfjobClient "github.com/kubeflow/tf-operator/pkg/client/clientset/versioned"
	"golang.org/x/net/context"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog"
)

type ExtensionServer struct {
	client tfjobClient.Interface
}

func (es *ExtensionServer) Release(ctx context.Context, in *queue.QueueUnit) error {
	name := in.Name
	namespace := in.Namespace
	tfjob, err := es.client.KubeflowV1().TFJobs(namespace).Get(name, metav1.GetOptions{})
	if err != nil {
		return err
	}

	var annotation map[string]string = map[string]string{}
	for k, v := range tfjob.Annotations {
		if k != queue.AnnotationQueue {
			annotation[k] = v
		}
	}
	tfjob.SetAnnotations(annotation)

	_, err = es.client.KubeflowV1().TFJobs(namespace).Update(tfjob)
	if err != nil {
		klog.Warningf("failed to released tf-job %s/%s\n", namespace, name)
	}
	return err
}

func MakeExtensionServer(client *tfjobClient.Clientset) communicate.ExtensionServerInterface {
	return &ExtensionServer{
		client: client,
	}
}

func StartServer(eci communicate.ExtensionServerInterface, unParsedAddr string) error {
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
			case http.MethodDelete:
				err = eci.Release(context.Background(), &qu)
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
		},
	)

	server := &http.Server{
		Handler: nil,
	}

	return server.Serve(lis)
}
