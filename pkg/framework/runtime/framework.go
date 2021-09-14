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

package runtime

import (
	"context"

	"k8s.io/client-go/informers"

	"github.com/kube-queue/api/pkg/client/clientset/versioned"
	"github.com/kube-queue/kube-queue/pkg/framework"
)

var _ framework.Framework = &frameworkImpl{}

type frameworkImpl struct {
	multiQueueSortPlugin   framework.MultiQueueSortPlugin
	filterPlugins          []framework.FilterPlugin
	queueSortPlugins       []framework.QueueSortPlugin
	reservePlugins         []framework.ReservePlugin
	kubeConfigPath         string
	sharedInformersFactory informers.SharedInformerFactory
	queueUnitClient        *versioned.Clientset
}

func (f *frameworkImpl) MultiQueueSortFunc() framework.MultiQueueLessFunc {
	return f.multiQueueSortPlugin.MultiQueueLess
}

func (f *frameworkImpl) QueueSortFuncMap() map[string]framework.QueueLessFunc {
	queueLessFuncMap := make(map[string]framework.QueueLessFunc)
	for _, plugin := range f.queueSortPlugins {
		queueLessFuncMap[plugin.Name()] = plugin.QueueLess
	}

	return queueLessFuncMap
}

func (f *frameworkImpl) RunFilterPlugins(ctx context.Context, unit *framework.QueueUnitInfo) *framework.Status {
	for _, pl := range f.filterPlugins {
		pluginStatus := pl.Filter(ctx, unit)
		if pluginStatus.Code() != framework.Success {
			return pluginStatus
		}
	}

	return framework.NewStatus(framework.Success, "")
}

func (f *frameworkImpl) RunScorePlugins(ctx context.Context) (int64, bool) {
	return 0, false
}

func (f *frameworkImpl) RunReservePluginsReserve(ctx context.Context, unit *framework.QueueUnitInfo) *framework.Status {
	for _, pl := range f.reservePlugins {
		pluginStatus := pl.Reserve(ctx, unit)
		if pluginStatus.Code() != framework.Success {
			return pluginStatus
		}
	}

	return framework.NewStatus(framework.Success, "")
}

func (f *frameworkImpl) RunReservePluginsUnreserve(ctx context.Context, unit *framework.QueueUnitInfo) {
	for _, pl := range f.reservePlugins {
		pl.Unreserve(ctx, unit)
	}
}

func (f *frameworkImpl) SharedInformerFactory() informers.SharedInformerFactory {
	return f.sharedInformersFactory
}

func (f *frameworkImpl) KubeConfigPath() string {
	return f.kubeConfigPath
}

func (f *frameworkImpl) QueueUnitClient() *versioned.Clientset {
	return f.queueUnitClient
}

func NewFramework(r Registry, kubeConfigPath string,
	informersFactory informers.SharedInformerFactory,
	queueUnitClient *versioned.Clientset,
) (framework.Framework, error) {
	filterPlugins := make([]framework.FilterPlugin, 0)
	queueSortPlugins := make([]framework.QueueSortPlugin, 0)
	reservePlugins := make([]framework.ReservePlugin, 0)
	var multiQueueSortPlugin framework.MultiQueueSortPlugin

	f := &frameworkImpl{
		kubeConfigPath:         kubeConfigPath,
		sharedInformersFactory: informersFactory,
		queueUnitClient:        queueUnitClient,
	}

	for _, factory := range r {
		p, err := factory(nil, f)
		if err != nil {
			return nil, err
		}
		if i, ok := p.(framework.QueueSortPlugin); ok {
			queueSortPlugins = append(queueSortPlugins, i)
		}
		if i, ok := p.(framework.MultiQueueSortPlugin); ok {
			multiQueueSortPlugin = i
		}
		if i, ok := p.(framework.FilterPlugin); ok {
			filterPlugins = append(filterPlugins, i)
		}
		if i, ok := p.(framework.ReservePlugin); ok {
			reservePlugins = append(reservePlugins, i)
		}
	}

	f.queueSortPlugins = queueSortPlugins
	f.reservePlugins = reservePlugins
	f.multiQueueSortPlugin = multiQueueSortPlugin
	f.filterPlugins = filterPlugins

	return f, nil
}
