package controller

import (
	"fmt"
	"time"

	queue "github.com/kube-queue/kube-queue/pkg/apis/queue/v1alpha1"
	extension "github.com/kube-queue/kube-queue/pkg/comm/extension"
	permission "github.com/kube-queue/kube-queue/pkg/comm/permission"
	corev1 "k8s.io/api/core/v1"
	utilrntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"
)

type Controller struct {
	// workqueue is the where queue units are stored
	WorkQueue workqueue.RateLimitingInterface

	// Permission Counter Client
	PClient permission.PermissionClientInterface

	// Extension Client Map
	ExtensionClients map[string]extension.ExtensionClientInterface

	// A Cache for Queue Unit
	qs map[string]queue.QueueUnit

	// recorder is an event recorder for recording Event resources to the
	// Kubernetes API.
	recorder record.EventRecorder
}

func NewController(
	kubeclientset kubernetes.Interface,
	pclient permission.PermissionClientInterface,
	extClients map[string]extension.ExtensionClientInterface) (*Controller, error) {
	// Create event broadcaster
	// Add sample-controller types to the default Kubernetes Scheme so Events can be
	// logged for sample-controller types.
	klog.V(4).Info("Creating event broadcaster")
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(klog.Infof)
	eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: kubeclientset.CoreV1().Events("")})

	schemeModified := scheme.Scheme
	recorder := eventBroadcaster.NewRecorder(schemeModified, corev1.EventSource{Component: controllerAgentName})

	controller := &Controller{
		WorkQueue: workqueue.NewNamedRateLimitingQueue(
			workqueue.NewItemFastSlowRateLimiter(2*time.Second, 1*time.Minute, 50), "priority"),
		PClient:          pclient,
		ExtensionClients: extClients,
		qs:               map[string]queue.QueueUnit{},
		recorder:         recorder,
	}

	klog.Info("Setting up event handlers")

	return controller, nil
}

func (c *Controller) EnqueueItem(qu *queue.QueueUnit) {
	key := qu.Serialize()
	if qu.Status.Phase == queue.JobEnqueued {
		c.qs[key] = *qu.DeepCopy()
		c.WorkQueue.AddRateLimited(key)
	}
}

func (c *Controller) DequeueItem(namespace string, name string, uid string, jobType string) error {
	qu := queue.MakeSimpleQueueUnit(name, namespace, uid, jobType)
	key := qu.Serialize()
	if err := c.PClient.NotifyDeleted(qu); err != nil {
		return err
	}

	c.WorkQueue.Forget(key)
	delete(c.qs, key)
	return nil
}

func (c *Controller) Run(threadiness int, stopCh <-chan struct{}) error {
	defer utilrntime.HandleCrash()
	defer c.WorkQueue.ShutDown()

	// Start the informer factories to begin populating the informer caches
	klog.Info("Starting Kube-Queue controller")

	// Wait for the caches to be synced before starting workers
	klog.Info("Waiting for informer caches to sync")
	var syncs []cache.InformerSynced
	// TODO: add all synced needed to syncs
	if ok := cache.WaitForCacheSync(stopCh, syncs...); !ok {
		return fmt.Errorf("failed to wait for caches to sync")
	}

	klog.Info("Starting workers")
	// Launch two workers to process Foo resources
	for i := 0; i < threadiness; i++ {
		go wait.Until(c.runWorker, time.Second, stopCh)
	}

	klog.Info("Started workers")
	<-stopCh
	klog.Info("Shutting down workers")

	return nil
}

func (c *Controller) runWorker() {
	for c.processNextWorkItem() {
	}
}

func (c *Controller) processNextWorkItem() bool {
	obj, shutdown := c.WorkQueue.Get()
	if shutdown {
		return false
	}

	defer c.WorkQueue.Done(obj)

	var key string
	var ok bool
	if key, ok = obj.(string); !ok {
		c.WorkQueue.Forget(obj)
		utilrntime.HandleError(fmt.Errorf("expected string in workqueue but got %#v", obj))
		return true
	}

	forget, err := c.syncHandler(key)
	if err == nil {
		if forget {
			c.WorkQueue.Forget(key)
		}
		return true
	}

	utilrntime.HandleError(err)
	c.WorkQueue.AddRateLimited(key)

	return true
}

func (c *Controller) syncHandler(key string) (bool, error) {
	klog.Infof("Processing key: %s", key)

	qu, ok := c.qs[key]
	if !ok {
		return true, fmt.Errorf("cannot find key: %s from internal QueueUnit cache", key)
	}

	permitted, err := c.PClient.IfPermissionReceived(&qu)
	if err != nil {
		return false, err
	}
	if !permitted {
		return false, fmt.Errorf("permission denied for job %s", key)
	}

	jobType := qu.Spec.JobType
	extClient, exist := c.ExtensionClients[jobType]
	if !exist {
		return false, fmt.Errorf("cannot find the corresponding extension client for %s %s", jobType, key)
	}

	err = extClient.ReleaseJob(&qu)
	if err != nil {
		return false, err
	}

	return true, nil
}
