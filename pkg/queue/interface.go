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
	// AddUnschedulableIfNotPresent inserts a queue unit that cannot be scheduled into
	// the queue, unless it is already in the queue. If there has been a recent move
	// request, then the queue unit is put in `podBackoffQ`.
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
