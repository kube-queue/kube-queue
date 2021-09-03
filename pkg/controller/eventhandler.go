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

package controller

import (
	"context"

	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"

	"github.com/kube-queue/api/pkg/apis/scheduling/v1alpha1"
	"github.com/kube-queue/kube-queue/pkg/framework"
)

func (c *Controller) addAllEventHandlers(queueInformer cache.SharedIndexInformer) {
	queueInformer.AddEventHandler(
		cache.FilteringResourceEventHandler{
			FilterFunc: func(obj interface{}) bool {
				switch qu := obj.(type) {
				case *v1alpha1.QueueUnit:
					if qu.Status.Phase != v1alpha1.Dequeued {
						return true
					}
					return false
				default:
					return false
				}
			},
			Handler: cache.ResourceEventHandlerFuncs{
				AddFunc:    c.AddQueueUnit,
				UpdateFunc: c.UpdateQueueUnit,
				DeleteFunc: c.DeleteQueueUnit,
			},
		},
	)

	queueInformer.AddEventHandler(
		cache.FilteringResourceEventHandler{
			FilterFunc: func(obj interface{}) bool {
				switch qu := obj.(type) {
				case *v1alpha1.QueueUnit:
					if qu.Status.Phase == v1alpha1.Dequeued {
						return true
					}
					return false
				default:
					return false
				}
			},
			Handler: cache.ResourceEventHandlerFuncs{
				AddFunc:    c.AddDequeuedQueueUnit,
				DeleteFunc: c.DeleteDequeuedQueueUnit,
			},
		},
	)
}

func (c *Controller) AddQueueUnit(obj interface{}) {
	unit := obj.(*v1alpha1.QueueUnit)
	queueName := unit.Spec.Queue
	q, ok := c.multiSchedulingQueue.GetQueueByName(queueName)
	if !ok {
		klog.Errorf("queue is not exist %s", queueName)
		return
	}

	err := q.Add(unit)
	if err != nil {
		klog.Errorf("queue %s add unit fail %v", queueName, err.Error())
	}
}

func (c *Controller) AddDequeuedQueueUnit(obj interface{}) {
	unit := obj.(*v1alpha1.QueueUnit)
	// TODO add reserveIfNotPresent
	c.fw.RunReservePluginsReserve(context.TODO(), framework.NewQueueUnitInfo(unit))
}

func (c *Controller) DeleteQueueUnit(obj interface{}) {
	unit := obj.(*v1alpha1.QueueUnit)
	queueName := unit.Spec.Queue
	q, ok := c.multiSchedulingQueue.GetQueueByName(queueName)
	if !ok {
		klog.Errorf("queue is not exist %s", queueName)
		return
	}

	err := q.Delete(unit)
	if err != nil {
		klog.Errorf("queue %s delete unit fail %v", queueName, err.Error())
	}
}

func (c *Controller) DeleteDequeuedQueueUnit(obj interface{}) {
	unit := obj.(*v1alpha1.QueueUnit)
	// TODO add unreserveIfNotPresent
	c.fw.RunReservePluginsUnreserve(context.TODO(), framework.NewQueueUnitInfo(unit))
}

func (c *Controller) UpdateQueueUnit(oldObj, newObj interface{}) {
	oldQu := oldObj.(*v1alpha1.QueueUnit)
	newQu := newObj.(*v1alpha1.QueueUnit)
	queueName := newQu.Spec.Queue
	q, ok := c.multiSchedulingQueue.GetQueueByName(queueName)
	if !ok {
		klog.Errorf("queue is not exist %s", queueName)
		return
	}

	err := q.Update(oldQu, newQu)
	if err != nil {
		klog.Errorf("queue %s update unit fail %v", queueName, err.Error())
	}
}
