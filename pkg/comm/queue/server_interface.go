package communicate

import (
	queue "github.com/kube-queue/kube-queue/pkg/apis/queue/v1alpha1"
	"golang.org/x/net/context"
)

type QueueServerInterface interface {
	AddOrUpdate(ctx context.Context, in *queue.QueueUnit) error
	Delete(ctx context.Context, in *queue.QueueUnit) error
}
