package framework

import (
	"time"

	"github.com/kube-queue/api/pkg/apis/scheduling/v1alpha1"
)

// QueueInfo is a Queue wrapper with additional information related to the Queue
type QueueInfo struct {
	Name     string
	Priority int32
	Queue    *v1alpha1.Queue
}

// QueueUnitInfo is a Queue wrapper with additional information related to
// the QueueUnit
type QueueUnitInfo struct {
	// Name is namespace + "/" + name
	Name string
	Unit *v1alpha1.QueueUnit
	// The time QueueUnit added to the scheduling queue.
	Timestamp time.Time
	// Number of schedule attempts before successfully scheduled.
	Attempts int
	// The time when the QueueUnit is added to the queue for the first time.
	InitialAttemptTimestamp time.Time
}

// NewQueueUnitInfo constructs QueueUnitInfo
func NewQueueUnitInfo(unit *v1alpha1.QueueUnit) *QueueUnitInfo {
	return &QueueUnitInfo{
		Name:                    unit.Namespace + "/" + unit.Name,
		Unit:                    unit,
		Timestamp:               time.Now(),
		Attempts:                0,
		InitialAttemptTimestamp: time.Now(),
	}
}
