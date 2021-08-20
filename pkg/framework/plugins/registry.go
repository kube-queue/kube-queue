package plugins

import (
	"github.com/kube-queue/kube-queue/pkg/framework/plugins/priority"
	"github.com/kube-queue/kube-queue/pkg/framework/plugins/resourcequota"
	"github.com/kube-queue/kube-queue/pkg/framework/runtime"
)

// NewInTreeRegistry builds the registry with all the in-tree plugins.
// A scheduler that runs out of tree plugins can register additional plugins
// through the WithFrameworkOutOfTreeRegistry option.
func NewInTreeRegistry() runtime.Registry {
	return runtime.Registry{
		resourcequota.Name: resourcequota.New,
		priority.Name:      priority.New,
	}
}
