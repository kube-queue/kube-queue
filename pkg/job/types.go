package job

import (
	"fmt"
	"strings"
)

const Separator = '/'

type GenericJob struct {
	Name      string
	Namespace string
	Kind      string
}

func (gj GenericJob) String() string {
	return fmt.Sprintf("%s%c%s%c%s",
		gj.Kind, Separator, gj.Namespace, Separator, gj.Name)
}

func ConvertToGenericJob(key string) (*GenericJob, error) {
	res := strings.Split(key, string(Separator))
	if len(res) != 3 {
		return nil, fmt.Errorf("failed to parse key: %s", key)
	}
	return &GenericJob{
		Name:      res[2],
		Namespace: res[1],
		Kind:      res[0],
	}, nil
}
