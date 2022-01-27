/*
 Copyright 2021 The Kube-Queue Authors.

 Licensed under the Apache License, Version 2.0 (the "License");
 you may not use this file except in compliance with the License.
 You may obtain a copy of the License at

     http://www.apache.org/licenses/LICENSE-2.0

 Unless required by applicable law or agreed to in writing, software
 distributed under the License is distributed on an "AS IS" BASIS,
 WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 See the License for the specific language governing permissions and
 limitations under the License.
*/

package framework

import (
	"time"

	"github.com/kube-queue/api/pkg/apis/scheduling/v1alpha1"
)

// QueueInfo is a Queue wrapper with additional information related to the Queue
type QueueInfo struct {
	// Name is namespace
	Name string
	Queue *v1alpha1.Queue
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

// NewQueueInfo constructs QueueInfo
func NewQueueInfo(queue *v1alpha1.Queue) *QueueInfo {
	return &QueueInfo{
		Name: queue.Namespace,
		Queue: queue,
	}
}
