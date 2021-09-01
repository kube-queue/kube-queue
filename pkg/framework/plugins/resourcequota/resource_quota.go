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

	"github.com/kube-queue/kube-queue/pkg/framework"
	"k8s.io/apimachinery/pkg/runtime"
)

// Name is the name of the plugin used in the plugin registry and configurations.
const Name = "ResourceQuota"

// ResourceQuota is a plugin that implements ResourceQuota filter.
type ResourceQuota struct{}

var _ framework.FilterPlugin = &ResourceQuota{}

// Name returns name of the plugin.
func (rq *ResourceQuota) Name() string {
	return Name
}

func (rq *ResourceQuota) Filter(ctx context.Context, QueueUnit *framework.QueueUnitInfo) *framework.Status {
	return framework.NewStatus(0, "")
}

// New initializes a new plugin and returns it.
func New(_ runtime.Object, handle framework.Handle) (framework.Plugin, error) {
	return &ResourceQuota{}, nil
}
