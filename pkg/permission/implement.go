package permission

import (
	queue "github.com/kube-queue/kube-queue/pkg/apis/queue/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	listerv1 "k8s.io/client-go/listers/core/v1"
	"k8s.io/klog"
)

type ResourcePermissionCounter struct {
	jobSetByNamespace map[string][]queue.QueueUnit
	quotaLister       listerv1.ResourceQuotaLister
}

func (rpc *ResourcePermissionCounter) IfPermissionGranted(namespace string, uid types.UID) bool {
	resourceRequest := rpc.getRequestedResource(namespace, uid)

	resourceReserved := rpc.getReservedResource(namespace)

	rq, err := rpc.quotaLister.ResourceQuotas(namespace).Get("default")
	if err != nil {
		klog.Warningf("failed to get default resource quota in ns %s: %v", namespace, err)
		return false
	}

	var resourceTotal corev1.ResourceList = map[corev1.ResourceName]resource.Quantity{
		"cpu":    rq.Spec.Hard.Cpu().DeepCopy(),
		"memory": rq.Spec.Hard.Memory().DeepCopy(),
	}

	if enoughResource(resourceRequest, resourceReserved, resourceTotal) {
		return true
	}

	return false
}

func (rpc *ResourcePermissionCounter) RegisterJob(name string, namespace string, uid types.UID, res corev1.ResourceList, phase queue.JobPhase) {
	qu := queue.QueueUnit{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			// Because we only use QueueUnit internally, we set the UID of the QueueUnit identical to the Job
			UID: uid,
		},
		Spec: queue.Spec{
			Priority: queue.DefaultPriority,
			Queue:    queue.DefaultQueueName,
			Resource: res,
		},
		Status: queue.Status{
			Phase: phase,
		},
	}

	q, _ := rpc.jobSetByNamespace[namespace]
	if q == nil {
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
	rpc.jobSetByNamespace[namespace] = q
}

func (rpc *ResourcePermissionCounter) UnregisterJob(namespace string, uid types.UID) {
	q, nsExist := rpc.jobSetByNamespace[namespace]
	if !nsExist {
		return
	}

	if q != nil {
		for idx, qu := range q {
			if qu.UID == uid {
				q = append(q[:idx], q[idx+1:]...)
			}
		}
		rpc.jobSetByNamespace[namespace] = q
	}
}

func (rpc *ResourcePermissionCounter) MarkJobDequeued(namespace string, uid types.UID) {
	q, exist := rpc.jobSetByNamespace[namespace]
	if !exist {
		return
	}

	for idx, qu := range q {
		if qu.UID == uid {
			q[idx].Status.Phase = queue.JobDequeued
		}
	}
}

func MakeResourcePermissionCounter(lister listerv1.ResourceQuotaLister) CounterInterface {
	return &ResourcePermissionCounter{
		jobSetByNamespace: map[string][]queue.QueueUnit{},
		quotaLister:       lister,
	}
}

func enoughResource(jobResource corev1.ResourceList, reserved corev1.ResourceList, quota corev1.ResourceList) bool {
	for rType, rQuantity := range jobResource {
		// If resource type not defined, then prohibit it
		quotaQuantity, exist := quota[rType]
		if !exist {
			return false
		}

		quotaCopy := quotaQuantity.DeepCopy()
		// calculate remaining quantity
		if reservedQuantity, exist := reserved[rType]; exist {
			quotaCopy.Sub(reservedQuantity)
		}

		// finally calculate if the remaining value is enough for the job
		quotaCopy.Sub(rQuantity)
		if quotaCopy.Sign() == -1 {
			return false
		}
	}
	return true
}

func (rpc ResourcePermissionCounter) getReservedResource(namespace string) corev1.ResourceList {
	q, exist := rpc.jobSetByNamespace[namespace]
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

func (rpc ResourcePermissionCounter) getRequestedResource(namespace string, uid types.UID) corev1.ResourceList {
	q, exist := rpc.jobSetByNamespace[namespace]
	if !exist {
		return nil
	}

	for _, qu := range q {
		if qu.UID == uid {
			return qu.Spec.Resource
		}
	}

	return nil
}
