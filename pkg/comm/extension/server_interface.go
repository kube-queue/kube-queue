package communicate

import (
	queue "github.com/kube-queue/kube-queue/pkg/apis/queue/v1alpha1"
	"golang.org/x/net/context"
)

type ExtensionServerInterface interface {
	Release(ctx context.Context, in *queue.QueueUnit) error
}
