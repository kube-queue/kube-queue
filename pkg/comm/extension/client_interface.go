package communicate

import queue "github.com/kube-queue/kube-queue/pkg/apis/queue/v1alpha1"

type ExtensionClientInterface interface {
	DequeueJob(qu *queue.QueueUnit) error
	Close() error
}
