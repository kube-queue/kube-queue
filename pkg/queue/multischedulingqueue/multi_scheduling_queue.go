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

package multischedulingqueue

import (
	"sort"
	"sync"

	"github.com/kube-queue/kube-queue/pkg/utils"

	"github.com/kube-queue/kube-queue/pkg/framework/plugins/priority"

	"github.com/kube-queue/api/pkg/apis/scheduling/v1alpha1"
	"github.com/kube-queue/kube-queue/pkg/framework"
	"github.com/kube-queue/kube-queue/pkg/queue"
	"github.com/kube-queue/kube-queue/pkg/queue/schedulingqueue"
)

// Making sure that MultiSchedulingQueue implements MultiSchedulingQueue.
var _ queue.MultiSchedulingQueue = &MultiSchedulingQueue{}

type MultiSchedulingQueue struct {
	sync.RWMutex
	fw       framework.Framework
	queueMap map[string]queue.SchedulingQueue
	lessFunc framework.MultiQueueLessFunc
	podInitialBackoffSeconds int
	podMaxBackoffSeconds int
}

func NewMultiSchedulingQueue(fw framework.Framework, podInitialBackoffSeconds int, podMaxBackoffSeconds int) (queue.MultiSchedulingQueue, error) {

	mq := &MultiSchedulingQueue{
		fw:       fw,
		queueMap: make(map[string]queue.SchedulingQueue),
		lessFunc: fw.MultiQueueSortFunc(),
		podInitialBackoffSeconds: podInitialBackoffSeconds,
		podMaxBackoffSeconds: podMaxBackoffSeconds,
	}

	// TODO support mutil queue
	// 当前只是创建default
	defaultQueue := schedulingqueue.NewPrioritySchedulingQueue(fw, utils.Default, priority.Name, podInitialBackoffSeconds, podMaxBackoffSeconds)
	mq.queueMap[utils.Default] = defaultQueue

	return mq, nil
}

func (mq *MultiSchedulingQueue) Run() {
	for _, q := range mq.queueMap {
		q.Run()
	}
}

func (mq *MultiSchedulingQueue) Close() {
	for _, q := range mq.queueMap {
		q.Close()
	}
}

func (mq *MultiSchedulingQueue) Add(q *v1alpha1.Queue) error {
	pq := schedulingqueue.NewPrioritySchedulingQueue(mq.fw, q.Name, "priority", mq.podInitialBackoffSeconds, mq.podMaxBackoffSeconds)
	mq.queueMap[pq.Name()] = pq
	return nil
}

func (mq *MultiSchedulingQueue) Delete(q *v1alpha1.Queue) error {
	delete(mq.queueMap, q.Name)
	return nil
}

func (mq *MultiSchedulingQueue) Update(old *v1alpha1.Queue, new *v1alpha1.Queue) error {
	pq := schedulingqueue.NewPrioritySchedulingQueue(mq.fw, new.Name, "priority", mq.podInitialBackoffSeconds, mq.podMaxBackoffSeconds)
	mq.queueMap[pq.Name()] = pq
	return nil
}

func (mq *MultiSchedulingQueue) GetQueueByName(name string) (queue.SchedulingQueue, bool) {
	if name == "" {
		return mq.queueMap["default"], true
	}
	q, ok := mq.queueMap[name]
	return q, ok
}

func (mq *MultiSchedulingQueue) SortedQueue() []queue.SchedulingQueue {
	len := len(mq.queueMap)
	unSortedQueue := make([]queue.SchedulingQueue, len)

	index := 0
	for _, q := range mq.queueMap {
		unSortedQueue[index] = q
		index++
	}

	sort.Slice(unSortedQueue, func(i, j int) bool {
		return mq.lessFunc(unSortedQueue[i].QueueInfo(), unSortedQueue[j].QueueInfo())
	})

	return unSortedQueue
}

func queueInfoKeyFunc(obj interface{}) (string, error) {
	q := obj.(queue.SchedulingQueue)
	return q.Name(), nil
}
