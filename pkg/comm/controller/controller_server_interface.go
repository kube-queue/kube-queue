package communicate

import (
	"github.com/kube-queue/kube-queue/pkg/comm/controller/pb"
	"golang.org/x/net/context"
)

type ControllerServerInterface interface {
	AddOrUpdate(ctx context.Context, in *pb.AddRequest) (*pb.AddResponse, error)
	Delete(ctx context.Context, in *pb.DeleteRequest) (*pb.DeleteResponse, error)
}
