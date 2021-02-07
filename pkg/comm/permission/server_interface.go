package communicate

import (
	"github.com/kube-queue/kube-queue/pkg/comm/permission/pb"
	"golang.org/x/net/context"
)

type PermissionServerInterface interface {
	DemandPermission(ctx context.Context, in *pb.PermissionRequest) (*pb.PermissionResponse, error)
	NotifyJobDeleted(ctx context.Context, in *pb.ChangeStatusRequest) (*pb.ChangeStatusResponse, error)
}
