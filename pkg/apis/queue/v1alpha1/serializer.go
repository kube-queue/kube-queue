package v1alpha1

import (
	"fmt"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const Separator = '/'

func (in *QueueUnit) Serialize() string {
	return fmt.Sprintf("%s%c%s%c%s",
		in.Spec.JobType, Separator, in.Namespace, Separator, in.Name)
}

func Deserialize(key string) (*QueueUnit, error) {
	res := strings.Split(key, string(Separator))
	if len(res) != 3 {
		return nil, fmt.Errorf("failed to parse key: %s", key)
	}
	return MakeSimpleQueueUnit(res[2], res[1], res[0]), nil
}

func MakeSimpleQueueUnit(name string, namespace string, jobType string) *QueueUnit {
	return &QueueUnit{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: Spec{
			JobType: jobType,
		},
	}
}