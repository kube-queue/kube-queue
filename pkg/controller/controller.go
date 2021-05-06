package controller

import (
	"fmt"
	"time"

	queue "github.com/kube-queue/kube-queue/pkg/apis/queue/v1alpha1"
	extension "github.com/kube-queue/kube-queue/pkg/comm/extension"
	"github.com/kube-queue/kube-queue/pkg/permission"
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

	// Permission Counter
	PC permission.CounterInterface

	// Extension Client Map
	ExtensionClients map[string]extension.ExtensionClientInterface

	// extCh is an ExtensionClients worker channel for calling RPC asynchronously
	extCh chan *queue.QueueUnit

	// recorder is an event recorder for recording Event resources to the
	// Kubernetes API.
	recorder record.EventRecorder
}

func NewController(
	kubeclientset kubernetes.Interface,
	pc permission.CounterInterface,
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
		PC:               pc,
		ExtensionClients: extClients,
		recorder:         recorder,
		extCh: make(chan *queue.QueueUnit),
	}

	klog.Info("Setting up event handlers")

	return controller, nil
}

// EnqueueItem insert QueueUnit into worker queue
func (c *Controller) EnqueueItem(qu *queue.QueueUnit) {
	phase := queue.JobEnqueued
	if qu.Status.Phase != "" {
		phase = qu.Status.Phase
	}
	c.PC.RegisterJob(qu.Name, qu.Namespace, qu.UID, qu.Spec.Resource, phase)

	key := qu.Serialize()
	if qu.Status.Phase == queue.JobEnqueued {
		c.WorkQueue.AddRateLimited(key)
	}
}

// DequeueItem removes QueueUnit from work queue and marks corresponding job as Dequeued
func (c *Controller) DequeueItem(qu *queue.QueueUnit) {
	c.extCh <- qu
}


// ReleaseItem is called to release the resource when job is removed
func (c *Controller) ReleaseItem(qu *queue.QueueUnit) error {
	key := qu.Serialize()

	c.WorkQueue.Forget(key)
	c.WorkQueue.Done(key)

	c.PC.UnregisterJob(qu.Namespace, qu.UID)

	return nil
}

func (c *Controller) Run(threadiness int, stopCh <-chan struct{}) error {
	defer utilrntime.HandleCrash()
	defer c.WorkQueue.ShutDown()
	defer close(c.extCh)

	// Start the informer factories to begin populating the informer caches
	klog.Info("Starting Kube-Queue controller")

	// Wait for the caches to be synced before starting workers
	klog.Info("Waiting for informer caches to sync")
	var syncs []cache.InformerSynced
	// TODO: add all synced needed to syncs
	if ok := cache.WaitForCacheSync(stopCh, syncs...); !ok {
		return fmt.Errorf("failed to wait for caches to sync")
	}

	// Launch extension worker to call RPC asynchronously
	for i := 0; i < threadiness; i++ {
		go func() {
			for {
				qu := <-c.extCh
				c.extensionWorker(qu)
			}
		}()
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

func (c *Controller) extensionWorker(qu *queue.QueueUnit) {
	key := qu.Serialize()
	extClient, exist := c.ExtensionClients[qu.Spec.JobType]
	if !exist {
		klog.Errorf("cannot find the corresponding extension client for %s %s", qu.Spec.JobType, key)
		c.WorkQueue.AddRateLimited(key)
	}

	err := extClient.DequeueJob(qu)
	if err != nil {
		klog.Errorf("error calling extension %s DequeueJob method: %v", qu.Spec.JobType, err)
		c.WorkQueue.AddRateLimited(key)
	}

	c.PC.MarkJobDequeued(qu.Namespace, qu.UID)
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

	qu, err := queue.Deserialize(key)
	if err != nil {
		return true, fmt.Errorf("cannot find key: %s from internal QueueUnit cache", key)
	}

	if !c.PC.IfPermissionGranted(qu.Namespace, qu.UID) {
		return false, fmt.Errorf("permission denied for job %s", key)
	}

	c.DequeueItem(qu)

	return true, nil
}
