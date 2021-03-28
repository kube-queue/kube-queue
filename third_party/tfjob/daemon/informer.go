package daemon

import (
	queue "github.com/kube-queue/kube-queue/pkg/comm/queue"
	tfjobInformerv1 "github.com/kubeflow/tf-operator/pkg/client/informers/externalversions/tensorflow/v1"
	tfcontroller "github.com/kubeflow/tf-operator/pkg/controller.v1/tensorflow"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
)

type Daemon struct {
	client   queue.QueueClientInterface
	informer tfjobInformerv1.TFJobInformer
}

func MakeDaemon(cfg *rest.Config, addr string, namespace string) (*Daemon, error) {
	client, err := MakeQueueClient(addr)
	if err != nil {
		return nil, err
	}

	in := tfcontroller.NewUnstructuredTFJobInformer(cfg, namespace)
	in.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    client.AddFunc,
		DeleteFunc: client.DeleteFunc,
	})

	return &Daemon{
		client:   client,
		informer: in,
	}, nil
}

func (d *Daemon) Run(stopCh <-chan struct{}) {
	go d.informer.Informer().Run(stopCh)
}
