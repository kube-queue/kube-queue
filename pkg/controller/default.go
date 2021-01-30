package controller

import (
	"fmt"
	v1 "github.com/kubeflow/tf-operator/pkg/apis/tensorflow/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	metav1unstructured "k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
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

func extractFromUnstructured(obj interface{}) (metav1.Object, corev1.ResourceList, error) {
	un, ok := obj.(*metav1unstructured.Unstructured)
	if !ok {
		return nil, nil, fmt.Errorf("cannot convert object to Unstructured")
	}

	switch un.GetKind() {
	case v1.Kind:
		var tfjob v1.TFJob
		err := runtime.DefaultUnstructuredConverter.FromUnstructured(un.Object, &tfjob)
		if err != nil {
			return nil, nil, fmt.Errorf("cannot convert to tfjob")
		}
		return &tfjob, CalResourceRequestForTFJob(&tfjob), nil
	default:
		return nil, nil, fmt.Errorf("type %s is not supported", un.GetKind())
	}
}

