package communicate

import v1alpha1 "github.com/kube-queue/kube-queue/pkg/apis/queue/v1alpha1"

type ExtensionClientInterface interface {
	ReleaseJob(qu *v1alpha1.QueueUnit) error
	Close() error
}
