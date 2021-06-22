package queue

import (
	"github.com/kube-queue/kube-queue/pkg/framework"

	schedv1alpha1 "github.com/kube-queue/api/pkg/apis/scheduling/v1alpha1"
)

// MultiSchedulingQueue is interface of Multi Scheduling Queue.
type MultiSchedulingQueue interface {
	Add(*schedv1alpha1.Queue) error
	Delete(*schedv1alpha1.Queue) error
	Update(*schedv1alpha1.Queue, *schedv1alpha1.Queue) error
	SortedQueue() []SchedulingQueue
	GetQueueByName(name string) (SchedulingQueue, bool)
	Run()
	Close()
}

// SchedulingQueue is interface of Single Scheduling Queue.
type SchedulingQueue interface {
	Add(*schedv1alpha1.QueueUnit) error
	AddUnschedulableIfNotPresent(*framework.QueueUnitInfo) error
	Delete(*schedv1alpha1.QueueUnit) error
	Update(*schedv1alpha1.QueueUnit, *schedv1alpha1.QueueUnit) error
	Pop() (*framework.QueueUnitInfo, error)
	Name() string
	QueueInfo() *framework.QueueInfo
	Length() int
	Run()
	Close()
}
