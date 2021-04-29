package permission

import (
	queue "github.com/kube-queue/kube-queue/pkg/apis/queue/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
)

type CounterInterface interface {
	IfPermissionGranted(namespace string, uid types.UID) bool
	RegisterJob(name string, namespace string, uid types.UID, res corev1.ResourceList, phase queue.JobPhase)
	UnregisterJob(namespace string, uid types.UID)
	MarkJobDequeued(namespace string, uid types.UID)
}
