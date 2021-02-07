package communicate

import (
	"github.com/kube-queue/kube-queue/pkg/comm/queue/pb"
	"golang.org/x/net/context"
)

type QueueServerInterface interface {
	AddOrUpdate(ctx context.Context, in *pb.AddRequest) (*pb.AddResponse, error)
	Delete(ctx context.Context, in *pb.DeleteRequest) (*pb.DeleteResponse, error)
}
