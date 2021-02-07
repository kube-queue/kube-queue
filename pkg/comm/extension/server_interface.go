package communicate

import (
	"github.com/kube-queue/kube-queue/pkg/comm/extension/pb"
	"golang.org/x/net/context"
)

type ExtensionServerInterface interface {
	Release(ctx context.Context, in *pb.ReleaseRequest) (*pb.ReleaseResponse, error)
}