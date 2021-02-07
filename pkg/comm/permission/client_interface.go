package communicate

import queue "github.com/kube-queue/kube-queue/pkg/apis/queue/v1alpha1"

type PermissionClientInterface interface {
	IfPermissionReceived(qu *queue.QueueUnit) (bool, error)
	NotifyDeleted(qu *queue.QueueUnit) error
	Close() error
}
