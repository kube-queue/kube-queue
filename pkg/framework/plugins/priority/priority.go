/*
Copyright 2021 The Kubernetes Authors.

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

package priority

import (
	"github.com/kube-queue/kube-queue/pkg/framework"
	"k8s.io/apimachinery/pkg/runtime"
)

// Name is the name of the plugin used in the plugin registry and configurations.
const Name = "Priority"

// Priority is a plugin that implements Priority plugin.
type Priority struct{}

var _ framework.MultiQueueSortPlugin = &Priority{}
var _ framework.QueueSortPlugin = &Priority{}

// Name returns name of the plugin.
func (p *Priority) Name() string {
	return Name
}

func (p *Priority) MultiQueueLess(q1 *framework.QueueInfo, q2 *framework.QueueInfo) bool {
	p1 := q1.Priority
	p2 := q2.Priority
	return p1 > p2
}

func (p *Priority) QueueLess(u1 *framework.QueueUnitInfo, u2 *framework.QueueUnitInfo) bool {
	var p1, p2 int32 = 0, 0

	if u1.Unit.Spec.Priority != nil {
		p1 = *(u1.Unit.Spec.Priority)
	}

	if u2.Unit.Spec.Priority != nil {
		p2 = *(u2.Unit.Spec.Priority)
	}
	return (p1 > p2) || (p1 == p2 && u1.InitialAttemptTimestamp.Before(u2.InitialAttemptTimestamp))
}

// New initializes a new plugin and returns it.
func New(_ runtime.Object, handle framework.Handle) (framework.Plugin, error) {
	return &Priority{}, nil
}
