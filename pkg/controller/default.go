package controller

import (
	v1 "github.com/kubeflow/tf-operator/pkg/apis/tensorflow/v1"
	corev1 "k8s.io/api/core/v1"
)

func CalResourceRequestForTFJob(j *v1.TFJob) corev1.ResourceList {
	resource := corev1.ResourceList{}
	for _, spec := range j.Spec.TFReplicaSpecs {
		replicas := int(*spec.Replicas)
		for _, c := range spec.Template.Spec.Containers {
			if c.Resources.Requests != nil {
				for resourceType, resourceQuantity := range c.Resources.Requests {
					for i := 0; i < replicas-1; i++ {
						resourceQuantity.Add(resourceQuantity)
					}
					oldQuantity, ok := resource[resourceType]
					if ok {
						resourceQuantity.Add(oldQuantity)
					}
					resource[resourceType] = resourceQuantity
				}
			}
		}
	}
	return resource
}

func EnoughResource(jobResource corev1.ResourceList, reserved corev1.ResourceList, quota corev1.ResourceList) bool {
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
