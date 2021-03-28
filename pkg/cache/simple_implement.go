package cache

import (
	queue "github.com/kube-queue/kube-queue/pkg/apis/queue/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
)

type InternalCache struct {
	core map[string][]queue.QueueUnit
}

func MakeInternalCache() JobCacheInterface {
	return &InternalCache{}
}

func (ic *InternalCache) AddOrUpdate(qu queue.QueueUnit) {
	namespace := qu.Namespace

	q, exist := ic.core[namespace]
	if !exist {
		q = append(q, qu)
	} else {
		registered := false
		for idx, item := range q {
			if item.UID == qu.UID {
				q[idx] = qu
				registered = true
			}
		}
		if !registered {
			q = append(q, qu)
		}
	}

	ic.core[namespace] = q
}

func (ic *InternalCache) Remove(namespace string, uid types.UID) {
	q, nsExist := ic.core[namespace]
	if !nsExist {
		return
	}
	if q == nil {
		return
	}

	for idx, item := range q {
		if item.UID == uid {
			q = append(q[:idx], q[idx+1:]...)
		}
	}

	ic.core[namespace] = q
}

func (ic *InternalCache) ReservedResource(namespace string) corev1.ResourceList {
	q, exist := ic.core[namespace]
	if !exist {
		return nil
	}

	reserved := corev1.ResourceList{}
	for _, qu := range q {
		if qu.Status.Phase == queue.JobDequeued {
			for resName, resVal := range qu.Spec.Resource {
				newVal := resVal.DeepCopy()
				oldVal, resExist := reserved[resName]
				if resExist {
					newVal.Add(oldVal)
				}
				reserved[resName] = newVal
			}
		}
	}

	return reserved
}

func (ic *InternalCache) UpdatePhase(namespace string, uid types.UID, newPhase queue.JobPhase) {
	q, exist := ic.core[namespace]
	if !exist {
		return
	}

	for idx, qu := range q {
		if qu.UID == uid {
			q[idx].Status.Phase = newPhase
			return
		}
	}
}
