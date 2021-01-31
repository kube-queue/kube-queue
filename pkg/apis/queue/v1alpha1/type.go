package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type QueueUnit struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata" protobuf:"bytes,1,name=metadata"`
	Spec              Spec   `json:"spec" protobuf:"bytes,2,name=spec"`
	Status            Status `json:"status,omitempty" protobuf:"bytes,3,opt,name=status"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type QueueUnitList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []QueueUnit `json:"items"`
}

const DefaultPriority int32 = 999
const DefaultQueueName string = "default"

type Spec struct {
	JobType  string              `json:"type" protobuf:"bytes,1,name=jobType"`
	Priority int32               `json:"priority,omitempty" protobuf:"varint,2,opt,name=priority"`
	Queue    string              `json:"queue,omitempty" protobuf:"bytes,3,opt,name=queue"`
	Resource corev1.ResourceList `json:"resource" protobuf:"bytes,4,name=resource"`
}

type JobPhase string

const (
	JobEnqueued JobPhase = "Enqueued"
	JobDequeued JobPhase = "Dequeued"
)

type Status struct {
	Phase   JobPhase `json:"phase" protobuf:"bytes,1,name=phase"`
	Message string   `json:"message,omitempty" protobuf:"bytes,2,opt,name=message"`
}
