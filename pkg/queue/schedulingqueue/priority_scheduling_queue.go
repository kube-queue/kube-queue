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

package schedulingqueue

import (
	"fmt"
	"sync"
	"time"

	"k8s.io/apimachinery/pkg/util/wait"

	"k8s.io/kubernetes/pkg/scheduler/util"

	"k8s.io/klog/v2"

	"github.com/kube-queue/api/pkg/apis/scheduling/v1alpha1"
	"github.com/kube-queue/kube-queue/pkg/framework"
	"github.com/kube-queue/kube-queue/pkg/queue"
	"github.com/kube-queue/kube-queue/pkg/queue/heap"
)

// Making sure that PrioritySchedulingQueue implements SchedulingQueue.
var _ queue.SchedulingQueue = &PrioritySchedulingQueue{}

type PrioritySchedulingQueue struct {
	sync.RWMutex
	name       string
	pluginName string
	fw         framework.Framework
	items      *heap.Heap
	backoffQ   *heap.Heap
	queue      *framework.QueueInfo
	clock      util.Clock
	// pod initial backoff duration.
	podInitialBackoffDuration time.Duration
	// pod maximum backoff duration.
	podMaxBackoffDuration time.Duration
	stop                  chan struct{}
	closed                bool
}

func NewPrioritySchedulingQueue(fw framework.Framework, name string, pluginName string) queue.SchedulingQueue {
	queueSortFuncMap := fw.QueueSortFuncMap()
	lessFn := queueSortFuncMap[pluginName]

	comp := func(queueUnitInfo1, queueUnitInfo2 interface{}) bool {
		quInfo1 := queueUnitInfo1.(*framework.QueueUnitInfo)
		quInfo2 := queueUnitInfo2.(*framework.QueueUnitInfo)
		return lessFn(quInfo1, quInfo2)
	}

	q := &PrioritySchedulingQueue{
		fw:                        fw,
		name:                      name,
		pluginName:                pluginName,
		items:                     heap.New(unitInfoKeyFunc, comp),
		podInitialBackoffDuration: 1 * time.Second,
		podMaxBackoffDuration:     4 * time.Second,
		clock:                     util.RealClock{},
	}

	q.backoffQ = heap.NewWithRecorder(unitInfoKeyFunc, q.podsCompareBackoffCompleted)
	return q
}

func (p *PrioritySchedulingQueue) Run() {
	go wait.Until(p.flushBackoffQCompleted, 1.0*time.Second, p.stop)
}

func (p *PrioritySchedulingQueue) Close() {
	p.Lock()
	defer p.Unlock()
	close(p.stop)
	p.closed = true
}

func (p *PrioritySchedulingQueue) Add(q *v1alpha1.QueueUnit) error {
	p.Lock()
	defer p.Unlock()

	info := framework.NewQueueUnitInfo(q)
	err := p.items.Add(info)
	if err != nil {
		klog.Infof("err %v", err)
	}
	return err
}

func (p *PrioritySchedulingQueue) AddUnschedulableIfNotPresent(quInfo *framework.QueueUnitInfo) error {
	p.Lock()
	defer p.Unlock()

	_, ok, _ := p.items.Get(quInfo)
	if ok {
		return nil
	}
	_, ok, _ = p.backoffQ.Get(quInfo)
	if ok {
		return nil
	}

	p.backoffQ.Add(quInfo)
	return nil
}

func (p *PrioritySchedulingQueue) Delete(q *v1alpha1.QueueUnit) error {
	p.Lock()
	defer p.Unlock()

	key := fmt.Sprintf("%v/%v", q.Namespace, q.Name)
	info, ok, _ := p.items.GetByKey(key)
	if ok {
		err := p.items.Delete(info)
		return err
	}

	info, ok, _ = p.backoffQ.GetByKey(key)
	if ok {
		err := p.backoffQ.Delete(info)
		return err
	}

	return nil
}

func (p *PrioritySchedulingQueue) Update(old *v1alpha1.QueueUnit, new *v1alpha1.QueueUnit) error {
	p.Lock()
	defer p.Unlock()

	newInfo := framework.NewQueueUnitInfo(new)
	key := fmt.Sprintf("%v/%v", new.Namespace, new.Name)
	_, ok, _ := p.items.GetByKey(key)
	if ok {
		err := p.items.Update(newInfo)
		return err
	}

	_, ok, _ = p.backoffQ.GetByKey(key)
	if ok {
		err := p.backoffQ.Update(newInfo)
		return err
	}
	return nil
}

func (p *PrioritySchedulingQueue) Pop() (*framework.QueueUnitInfo, error) {
	p.Lock()
	defer p.Unlock()

	obj, err := p.items.Pop()
	u := obj.(*framework.QueueUnitInfo)
	return u, err
}

func (p *PrioritySchedulingQueue) TopUnit() (*framework.QueueUnitInfo, error) {
	p.Lock()
	defer p.Unlock()

	if p.items.Len() > 0 {
		obj := p.items.List()[0]
		u := obj.(*framework.QueueUnitInfo)
		return u, nil
	}
	return nil, fmt.Errorf("queue is empty")
}

func (p *PrioritySchedulingQueue) Name() string {
	return p.name
}

func (p *PrioritySchedulingQueue) QueueInfo() *framework.QueueInfo {
	return p.queue
}

func (p *PrioritySchedulingQueue) Length() int {
	return p.items.Len()
}

func unitInfoKeyFunc(obj interface{}) (string, error) {
	unit := obj.(*framework.QueueUnitInfo)
	return unit.Name, nil
}

// flushBackoffQCompleted Moves all pods from backoffQ which have completed backoff in to activeQ
func (p *PrioritySchedulingQueue) flushBackoffQCompleted() {
	p.Lock()
	defer p.Unlock()
	for {
		rawQUInfo := p.backoffQ.Peek()
		if rawQUInfo == nil {
			return
		}

		qu := rawQUInfo.(*framework.QueueUnitInfo)
		boTime := p.getBackoffTime(qu)
		if boTime.After(p.clock.Now()) {
			return
		}
		_, err := p.backoffQ.Pop()
		if err != nil {
			klog.Errorf("Unable to pop pod %v from backoff queue despite backoff completion.", qu.Unit.Namespace+"/"+qu.Unit.Name)
			return
		}
		p.items.Add(rawQUInfo)
	}
}

// getBackoffTime returns the time that podInfo completes backoff
func (p *PrioritySchedulingQueue) getBackoffTime(info *framework.QueueUnitInfo) time.Time {
	duration := p.calculateBackoffDuration(info)
	backoffTime := info.Timestamp.Add(duration)
	return backoffTime
}

// calculateBackoffDuration is a helper function for calculating the backoffDuration
// based on the number of attempts the pod has made.
func (p *PrioritySchedulingQueue) calculateBackoffDuration(info *framework.QueueUnitInfo) time.Duration {
	duration := p.podInitialBackoffDuration
	for i := 1; i < info.Attempts; i++ {
		duration = duration * 2
		if duration > p.podMaxBackoffDuration {
			return p.podMaxBackoffDuration
		}
	}
	return duration
}

func (p *PrioritySchedulingQueue) podsCompareBackoffCompleted(quInfo1, quInfo2 interface{}) bool {
	info1 := quInfo1.(*framework.QueueUnitInfo)
	info2 := quInfo2.(*framework.QueueUnitInfo)
	bo1 := p.getBackoffTime(info1)
	bo2 := p.getBackoffTime(info2)
	return bo1.Before(bo2)
}
