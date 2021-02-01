package communicate

import (
	"fmt"
	"github.com/kube-queue/kube-queue/pkg/comm/controller/pb"
	"github.com/kube-queue/kube-queue/pkg/controller"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"net"
	"net/url"
)

type ControllerServer struct {
	queueController *controller.Controller
}

func MakeControllerServer(qc *controller.Controller) ControllerServerInterface {
	return &ControllerServer{queueController: qc}
}

func (cs *ControllerServer) AddOrUpdate(ctx context.Context, in *pb.AddRequest) (*pb.AddResponse, error) {
	queueUnit := in.QueueUnit.DeepCopy()
	cs.queueController.EnqueueItem(*queueUnit)
	return &pb.AddResponse{Message: "enqueued"}, nil
}

func (cs *ControllerServer) Delete(ctx context.Context, in *pb.DeleteRequest) (*pb.DeleteResponse, error) {
	cs.queueController.DequeueItem(in.Namespace, in.Name, in.Uid, in.Type)
	return &pb.DeleteResponse{Message: "dequeued"}, nil
}

func StartServer(csi ControllerServerInterface, unParsedAddr string) error {
	protocal, addr, err := parseEndpoint(unParsedAddr)
	if err != nil {
		return err
	}

	lis, err := net.Listen(protocal, addr)
	if err != nil {
		return err
	}

	s := grpc.NewServer()
	pb.RegisterQueueServer(s, csi)

	reflection.Register(s)

	err = s.Serve(lis)
	if err != nil {
		return err
	}

	return nil
}

func parseEndpoint(endpoint string) (string, string, error) {
	u, err := url.Parse(endpoint)
	if err != nil {
		return "", "", err
	}

	switch u.Scheme {
	case "tcp":
		return "tcp", u.Host, nil

	case "unix":
		return "unix", u.Path, nil

	case "":
		return "", "", fmt.Errorf("using %q as endpoint is deprecated, please consider using full url format", endpoint)

	default:
		return u.Scheme, "", fmt.Errorf("protocol %q not supported", u.Scheme)
	}
}