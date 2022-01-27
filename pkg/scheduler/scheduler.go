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

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"

	"github.com/kube-queue/api/pkg/apis/scheduling/v1alpha1"
	"github.com/kube-queue/api/pkg/client/clientset/versioned"
	"github.com/kube-queue/kube-queue/pkg/framework"
	"github.com/kube-queue/kube-queue/pkg/queue"
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

func (s *Scheduler) Start(ctx context.Context) {
	s.internalSchedule(ctx)
}

// Internal start scheduling
func (s *Scheduler) internalSchedule(ctx context.Context) {
	for {
		s.schedule(ctx)
	}
}

func (s *Scheduler) schedule(ctx context.Context) {
	schedulingCycleCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	sortedQueue := s.multiSchedulingQueue.SortedQueue()
	for _, q := range sortedQueue {
		if q.Length() > 0 {
			unitInfo, err := q.Pop()
			if err != nil {
				klog.Errorf("get topunit err %v", err)
			}
			klog.Info("---schedule begin %v ---", unitInfo.Name)
			status := s.fw.RunFilterPlugins(schedulingCycleCtx, unitInfo)
			klog.Info("filter status %v %v", status.Code(), status.Message())
			if status.Code() == framework.Success {
				klog.Infof("dequeue %v", unitInfo.Name)
				status = s.fw.RunReservePluginsReserve(schedulingCycleCtx, unitInfo)
				klog.Info("reserve status %v %v", status.Code(), status.Message())
				go func() {
					err := s.Dequeue(unitInfo.Unit)
					if err != nil {
						klog.Errorf("dequeue %v failed: %v", unitInfo.Name, err.Error())
						// 构建一个临时存储的位置
						s.fw.RunReservePluginsUnreserve(schedulingCycleCtx, unitInfo)
						s.ErrorFunc(ctx, unitInfo, q)
					}
					klog.Info("dequeue %v success", unitInfo.Name)
					klog.Info("---schedule end %v ---", unitInfo.Name)
				}()
			} else {
				s.ErrorFunc(ctx, unitInfo, q)
				klog.Info("---schedule end %v ---", unitInfo.Name)
			}
		}
	}
}

func (s *Scheduler) Dequeue(queueUnit *v1alpha1.QueueUnit) error {
	newQueueUnit, err := s.QueueClient.SchedulingV1alpha1().QueueUnits(queueUnit.Namespace).Get(context.TODO(), queueUnit.Name, metav1.GetOptions{})
	if err != nil {
		return err
	}

	newQueueUnit.Status.Phase = v1alpha1.Dequeued
	newQueueUnit.Status.Message = "Dequeued because schedule successfully"
	_, err = s.QueueClient.SchedulingV1alpha1().QueueUnits(queueUnit.Namespace).Update(context.TODO(), newQueueUnit, metav1.UpdateOptions{})
	if err != nil {
		return err
	}

	klog.Infof("%v/%v dequeue success", newQueueUnit.Namespace, newQueueUnit.Name)
	return nil
}

func (s *Scheduler) ErrorFunc(ctx context.Context, queueUnit *framework.QueueUnitInfo, q queue.SchedulingQueue) {
	queueUnit.Attempts++
	queueUnit.Timestamp = time.Now()
	newQueueUnit, err := s.QueueClient.SchedulingV1alpha1().QueueUnits(queueUnit.Unit.Namespace).Get(ctx, queueUnit.Unit.Name, v1.GetOptions{})
	if err != nil {
		klog.Errorf("get qu %v error %v", queueUnit.Name, err)
		return
	}
	queueUnit.Unit = newQueueUnit
	err = q.AddUnschedulableIfNotPresent(queueUnit)
	if err != nil {
		klog.Errorf("Add Unschedulable QueueUnit %v failed %v", queueUnit.Name)
	}
}
