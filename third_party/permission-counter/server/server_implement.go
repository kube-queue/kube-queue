package server

import (
	"fmt"
	"net"

	queue "github.com/kube-queue/kube-queue/pkg/apis/queue/v1alpha1"
	"github.com/kube-queue/kube-queue/pkg/cache"
	permission "github.com/kube-queue/kube-queue/pkg/comm/permission"
	"github.com/kube-queue/kube-queue/pkg/comm/permission/pb"
	"github.com/kube-queue/kube-queue/pkg/comm/utils"
	"github.com/kube-queue/kube-queue/pkg/controller"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type GRPCServer interface {
	Serve() error
}

type GRPCServerForPermissionCounter struct {
	server   *grpc.Server
	listener net.Listener
}

func (s *GRPCServerForPermissionCounter) Serve() error {
	return s.server.Serve(s.listener)
}

type PermissionServer struct {
	client             *kubernetes.Clientset
	namespacedJobCache cache.JobCacheInterface
}

func (p *PermissionServer) NotifyJobDeleted(ctx context.Context, in *pb.ChangeStatusRequest) (*pb.ChangeStatusResponse, error) {
	namespace := in.QueueUnit.Namespace
	uid := in.QueueUnit.UID
	p.namespacedJobCache.Remove(namespace, uid)
	return &pb.ChangeStatusResponse{Result: pb.ActionResult_SUCCESS}, nil
}

func (p *PermissionServer) DemandPermission(ctx context.Context, in *pb.PermissionRequest) (*pb.PermissionResponse, error) {
	namespace := in.QueueUnit.Namespace
	uid := in.QueueUnit.UID
	resourceReserved := p.namespacedJobCache.ReservedResource(namespace)
	resourceRequested := in.QueueUnit.Spec.Resource

	quotaList, err := p.client.CoreV1().ResourceQuotas(namespace).List(v1.ListOptions{})
	if err != nil {
		return &pb.PermissionResponse{
			Feedback: pb.PermissionFeedback_DENIED,
		}, err
	}
	if len(quotaList.Items) < 1 {
		return &pb.PermissionResponse{
			Feedback: pb.PermissionFeedback_DENIED,
		}, fmt.Errorf("cannot find any resource quota under namespace: %s", namespace)
	}
	resourceTotal := quotaList.Items[0].Spec.Hard

	if !controller.EnoughResource(resourceRequested, resourceReserved, resourceTotal) {
		return &pb.PermissionResponse{Feedback: pb.PermissionFeedback_DENIED}, nil
	}
	p.namespacedJobCache.UpdatePhase(namespace, uid, queue.JobDequeued)
	return &pb.PermissionResponse{Feedback: pb.PermissionFeedback_PROVE}, nil
}

func MakePermissionServer(client *kubernetes.Clientset) permission.PermissionServerInterface {
	return &PermissionServer{
		client:             client,
		namespacedJobCache: cache.MakeInternalCache(),
	}
}

func MakeGRPCServerForPermissionCounter(cfg *rest.Config, unParsedAddr string) (GRPCServer, error) {
	kubeClient, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}

	ps := MakePermissionServer(kubeClient)

	s := grpc.NewServer()
	pb.RegisterPermissionServer(s, ps)

	protocol, addr, err := utils.ParseEndpoint(unParsedAddr)
	if err != nil {
		return nil, err
	}

	lis, err := net.Listen(protocol, addr)
	if err != nil {
		return nil, err
	}

	return &GRPCServerForPermissionCounter{
		server:   s,
		listener: lis,
	}, nil
}
