package cache

import (
	queue "github.com/kube-queue/kube-queue/pkg/apis/queue/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
)

type JobCacheInterface interface {
	AddOrUpdate(unit queue.QueueUnit)
	Remove(namespace string, uid types.UID)
	ReservedResource(namespace string) corev1.ResourceList
	UpdatePhase(namespace string, uid types.UID, newPhase queue.JobPhase)
}
