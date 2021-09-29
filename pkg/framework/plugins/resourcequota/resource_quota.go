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

package resourcequota

import (
	"context"
	"fmt"
	"sync"

	"github.com/kube-queue/kube-queue/pkg/framework"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	clientcorev1 "k8s.io/client-go/listers/core/v1"
)

// Name is the name of the plugin used in the plugin registry and configurations.
const Name = "ResourceQuota"

const (
	ErrNoResourceQuotaTemplate            = "found %d resources quota in ns: %s, expecting more than 0"
	ErrNoProperResourceQuotaFoundTemplate = "cannot find proper resource quota in namespace %s"
	ErrResourceQuotaStatusHardNilTemplate = "cannot find hard limit in the status of resource quota %s"
	ErrResourceQuotaTypeNotFoundTemplate  = "resource type %s not found in resource quota %s"
	ErrResourceQuotaInsufficientTemplate  = "insufficient resource left for %s in resource quota %s reserved %v/%v, request %v"
	ErrQueueUnitAlreadyReservedTemplate   = "queue unit %s already reserved"
)

// ResourceQuota is a plugin that implements ResourceQuota filter.
type ResourceQuota struct {
	sync.RWMutex
	reserved map[string]corev1.ResourceList
	quRecord map[string]interface{}
	rqLister clientcorev1.ResourceQuotaLister
}

var _ framework.FilterPlugin = &ResourceQuota{}

// Name returns name of the plugin.
func (rq *ResourceQuota) Name() string {
	return Name
}

func QueueUnitToKey(qu *framework.QueueUnitInfo) string {
	return fmt.Sprintf("%s/%s", qu.Unit.GetNamespace(), qu.Unit.GetName())
}

// Reserve resource for the given QueueUnitInfo
func (rq *ResourceQuota) Reserve(ctx context.Context, qu *framework.QueueUnitInfo) *framework.Status {
	rq.Lock()
	defer rq.Unlock()

	key := QueueUnitToKey(qu)
	if _, exist := rq.quRecord[key]; exist {
		return framework.NewStatus(framework.Error, fmt.Sprintf(ErrQueueUnitAlreadyReservedTemplate, key))
	}

	ns := qu.Unit.Namespace
	reservedNS, exist := rq.reserved[ns]
	if !exist {
		reservedNS = make(corev1.ResourceList)
	}

	for rName, rQuantity := range qu.Unit.Spec.Resource {
		val, exist := reservedNS[rName]
		if exist {
			rQuantity.Add(val)
		}
		reservedNS[rName] = rQuantity
	}

	rq.reserved[ns] = reservedNS
	rq.quRecord[key] = nil

	return framework.NewStatus(framework.Success, "")
}

// Unreserve resource for the given QueueUnitInfo
func (rq *ResourceQuota) Unreserve(ctx context.Context, qu *framework.QueueUnitInfo) {
	rq.Lock()
	defer rq.Unlock()

	key := QueueUnitToKey(qu)
	if _, exist := rq.quRecord[key]; !exist {
		return
	}

	ns := qu.Unit.Namespace
	reservedNS, exist := rq.reserved[ns]
	if !exist {
		return
	}

	for rName, rQuantity := range qu.Unit.Spec.Resource {
		val, exist := reservedNS[rName]
		if !exist {
			continue
		}
		// resource quantity found
		val.Sub(rQuantity)
		if val.Sign() <= 0 {
			delete(reservedNS, rName)
			continue
		}
		reservedNS[rName] = val
	}

	rq.reserved[ns] = reservedNS
	delete(rq.quRecord, key)
}

// GetReservedByResourceName returns reserved resource quantity if the ResourceName is found,
// otherwise returns zero Quantity
func (rq *ResourceQuota) GetReservedByResourceName(ns string, rName corev1.ResourceName) resource.Quantity {
	rq.RLock()
	defer rq.RUnlock()

	reservedNS, exist := rq.reserved[ns]
	if !exist {
		reservedNS = make(corev1.ResourceList)
	}

	val, exist := reservedNS[rName]
	if exist {
		return val
	}

	return *resource.NewQuantity(0, resource.BinarySI)
}

// SelectResourceQuota returns the proper resource quota for the given namespace
func SelectResourceQuota(rqs []*corev1.ResourceQuota, ns string) (*corev1.ResourceQuota, error) {
	if len(rqs) == 0 {
		return nil, fmt.Errorf(ErrNoResourceQuotaTemplate, 0, ns)
	}
	// if there is only one resource quota, just return it
	if len(rqs) == 1 {
		return rqs[0], nil
	}
	// if there are multiple resource quota, select one that named as the namespace
	for _, rq := range rqs {
		if rq.GetName() == ns {
			return rq, nil
		}
	}
	return nil, fmt.Errorf(ErrNoProperResourceQuotaFoundTemplate, ns)
}

// Filter returns Status with success if there are enough resource left for the given QueueUnitInfo
func (rq *ResourceQuota) Filter(ctx context.Context, qu *framework.QueueUnitInfo) *framework.Status {
	// TODO: maybe there is a nil when locating the namespace; validate this QueueUnit first
	ns := qu.Unit.Spec.ConsumerRef.Namespace

	// Locate corresponding ResourceQuota
	rqs, err := rq.rqLister.ResourceQuotas(ns).List(labels.Everything())
	if err != nil {
		return framework.NewStatus(framework.Error, err.Error())
	}
	basket, err := SelectResourceQuota(rqs, ns)
	if err != nil {
		return framework.NewStatus(framework.Error, err.Error())
	}
	if basket.Spec.Hard == nil {
		return framework.NewStatus(framework.Error, fmt.Sprintf(ErrResourceQuotaStatusHardNilTemplate, basket.GetName()))
	}

	// Check if there are enough resource quota left for this unit
	for rName, rQuantity := range qu.Unit.Spec.Resource {
		basketQuantity, found := basket.Spec.Hard[rName]
		if !found {
			continue
		}
		reservedQuantity := rq.GetReservedByResourceName(ns, rName)
		reservedQuantity.Add(rQuantity)
		if basketQuantity.Cmp(reservedQuantity) < 0 {
			return framework.NewStatus(framework.Error,
				fmt.Sprintf(ErrResourceQuotaInsufficientTemplate, rName, basket.GetName(), reservedQuantity.Value(), basketQuantity.Value(), rQuantity.Value()))
		}
	}

	return framework.NewStatus(framework.Success, "")
}

// New initializes a new plugin and returns it.
func New(_ runtime.Object, handle framework.Handle) (framework.Plugin, error) {
	return &ResourceQuota{
		rqLister: handle.SharedInformerFactory().Core().V1().ResourceQuotas().Lister(),
		reserved: make(map[string]corev1.ResourceList),
		quRecord: make(map[string]interface{}),
	}, nil
}
