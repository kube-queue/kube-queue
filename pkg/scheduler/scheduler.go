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

package scheduler

import (
	"context"
	"time"

	"github.com/kube-queue/api/pkg/apis/scheduling/v1alpha1"
	"github.com/kube-queue/api/pkg/client/clientset/versioned"
	"github.com/kube-queue/kube-queue/pkg/framework"
	"github.com/kube-queue/kube-queue/pkg/queue"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
)

type Scheduler struct {
	multiSchedulingQueue queue.MultiSchedulingQueue
	fw                   framework.Framework
	QueueClient          *versioned.Clientset
}

func NewScheduler(multiSchedulingQueue queue.MultiSchedulingQueue, fw framework.Framework, queueClient *versioned.Clientset) (*Scheduler, error) {
	sche := &Scheduler{
		multiSchedulingQueue: multiSchedulingQueue,
		fw:                   fw,
		QueueClient:          queueClient,
	}
	return sche, nil
}

func (s *Scheduler) Start() {
	s.internalSchedule()
}

// Internal start scheduling
func (s *Scheduler) internalSchedule() {
	for {
		s.schedule()
	}
}

func (s *Scheduler) schedule() {
	sortedQueue := s.multiSchedulingQueue.SortedQueue()
	for _, q := range sortedQueue {
		if q.Length() > 0 {
			klog.Info("schedule cycle")
			unitInfo, err := q.Pop()
			if err != nil {
				klog.Errorf("get topunit err %v", err)
			}
			status := s.fw.RunFilterPlugins(unitInfo)
			if status.Code() == framework.Success {
				klog.Infof("dequeue %v", unitInfo.Name)
				s.fw.RunReservePluginsReserve(unitInfo)

				go func() {
					err := s.Dequeue(unitInfo.Unit)
					if err != nil {
						klog.Errorf("%v Dequeue failed %v", unitInfo.Name, err.Error())
						// 构建一个临时存储的位置
						s.fw.RunReservePluginsUnreserve(unitInfo)
						s.ErrorFunc(unitInfo, q)
					}
				}()
			} else {
				s.ErrorFunc(unitInfo, q)
			}
			return
		}
	}
}

func (s *Scheduler) Dequeue(u *v1alpha1.QueueUnit) error {
	_, err := s.QueueClient.SchedulingV1alpha1().QueueUnits(u.Namespace).Get(context.TODO(), u.Name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return err
	}

	u.Status.Phase = v1alpha1.Dequeued
	u.Status.Message = "Dequeued because schedule successfully"
	_, err = s.QueueClient.SchedulingV1alpha1().QueueUnits(u.Namespace).Update(context.TODO(), u, metav1.UpdateOptions{})
	if err != nil {
		return err
	}

	klog.Infof("%v/%v dequeue success", u.Namespace, u.Name)
	return nil
}

func (s *Scheduler) ErrorFunc(qu *framework.QueueUnitInfo, q queue.SchedulingQueue) {
	qu.Attempts++
	qu.Timestamp = time.Now()
	err := q.AddUnschedulableIfNotPresent(qu)
	if err != nil {
		klog.Errorf("Add Unschedulable QueueUnit %v failed %v", qu.Name)
	}
}
