package communicate

import (
	"net"

	"github.com/kube-queue/kube-queue/pkg/comm/queue/pb"
	"github.com/kube-queue/kube-queue/pkg/comm/utils"
	"github.com/kube-queue/kube-queue/pkg/controller"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

type QueueServer struct {
	queueController *controller.Controller
}

func MakeQueueServer(qc *controller.Controller) QueueServerInterface {
	return &QueueServer{queueController: qc}
}

func (cs *QueueServer) AddOrUpdate(ctx context.Context, in *pb.AddRequest) (*pb.AddResponse, error) {
	queueUnit := in.QueueUnit.DeepCopy()
	cs.queueController.EnqueueItem(queueUnit)
	return &pb.AddResponse{Message: "enqueued"}, nil
}

func (cs *QueueServer) Delete(ctx context.Context, in *pb.DeleteRequest) (*pb.DeleteResponse, error) {
	err := cs.queueController.DequeueItem(in.Namespace, in.Name, in.Uid, in.Type)
	if err != nil {
		return nil, err
	}
	return &pb.DeleteResponse{Message: "dequeued"}, nil
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

	s := grpc.NewServer()
	pb.RegisterQueueServer(s, csi)

	reflection.Register(s)

	err = s.Serve(lis)
	if err != nil {
		return err
	}

	return nil
}
