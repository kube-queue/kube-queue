package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type QueueUnit struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`
	Spec              Spec   `json:"spec"`
	Status            Status `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type QueueUnitList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []QueueUnit `json:"items"`
}

type Spec struct {
	JobType  string              `json:"type"`
	Priority int32               `json:"priority"`
	Queue    string              `json:"queue"`
	Resource corev1.ResourceList `json:"resource"`
}

type JobPhase string

const (
	JobEnqueued JobPhase = "Enqueued"
	JobDequeued JobPhase = "Dequeued"
)

type Status struct {
	Phase   JobPhase `json:"phase"`
	Message string   `json:"message,omitempty"`
}
