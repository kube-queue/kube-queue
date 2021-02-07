package server

import (
	communicate "github.com/kube-queue/kube-queue/pkg/comm/extension"
	"github.com/kube-queue/kube-queue/pkg/comm/extension/pb"
	"github.com/kube-queue/kube-queue/pkg/comm/utils"
	tfjobClient "github.com/kubeflow/tf-operator/pkg/client/clientset/versioned"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"net"
)

type GRPCServer interface {
	Serve() error
}

type GRPCServerForTFJob struct {
	server *grpc.Server
	listener net.Listener
}

func (s *GRPCServerForTFJob) Serve() error {
	return s.server.Serve(s.listener)
}

type ExtensionServer struct {
	client *tfjobClient.Clientset
}

func (es *ExtensionServer) Release(ctx context.Context, in *pb.ReleaseRequest) (*pb.ReleaseResponse, error) {
	name := in.Name
	namespace := in.Namespace
	err := es.client.KubeflowV1().TFJobs(namespace).Delete(name, &metav1.DeleteOptions{})
	if err != nil {
		return nil, err
	}
	return &pb.ReleaseResponse{Message: "released"}, nil
}

func MakeExtensionServer(client *tfjobClient.Clientset) communicate.ExtensionServerInterface {
	return &ExtensionServer{
		client: client,
	}
}

func MakeGRPCServerForTFJob(cfg *rest.Config, unParsedAddr string) (GRPCServer, error) {
	client, err := tfjobClient.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}

	es := MakeExtensionServer(client)

	s := grpc.NewServer()
	pb.RegisterJobServer(s, es)

	reflection.Register(s)

	protocol, addr, err := utils.ParseEndpoint(unParsedAddr)
	if err != nil {
		return nil, err
	}

	lis, err := net.Listen(protocol, addr)
	if err != nil {
		return nil, err
	}

	return &GRPCServerForTFJob{
		server:   s,
		listener: lis,
	}, nil
}