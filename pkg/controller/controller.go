package controller

import (
	"fmt"
	"strings"
	"time"

	"github.com/kube-queue/kube-queue/cmd/app/options"
	queue "github.com/kube-queue/kube-queue/pkg/apis/queue/v1alpha1"
	v1 "github.com/kubeflow/tf-operator/pkg/apis/tensorflow/v1"
	tfjob "github.com/kubeflow/tf-operator/pkg/client/clientset/versioned"
	"github.com/kubeflow/tf-operator/pkg/client/informers/externalversions"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"
)

const controllerAgentName = "queue-controller"

const (
	annotationQueue = "kube-queue"
)

type Controller struct {
	// kubeclientset is a standard kubernetes clientset
	kubeclientset kubernetes.Interface
	// jobClientset is a map of job type: clientset for enabled job type
	jobClientsets map[string]interface{}
	// jobInformers is a map of job type: informer for enabled job type
	jobInformers map[string]interface{}

	// workqueue is the where queue units are stored
	// TODO 1: extend single-queue to multi-queue
	// TODO 2: the default workqueue is not compatible with a comparing function, need to extend workqueue for
	// TODO  : priority queueing
	workqueue workqueue.RateLimitingInterface
	// unitMap maps jobtype-namespace-name to the Queue Unit
	unitMap map[string]*queue.QueueUnit

	// recorder is an event recorder for recording Event resources to the
	// Kubernetes API.
	recorder record.EventRecorder
}

func NewController(kubeclientset kubernetes.Interface,
	jobClientsets map[string]interface{},
	jobInformers map[string]interface{},
	stopCh <-chan struct{},
) (*Controller, error) {
	// Create event broadcaster
	// Add sample-controller types to the default Kubernetes Scheme so Events can be
	// logged for sample-controller types.
	klog.V(4).Info("Creating event broadcaster")
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(klog.Infof)
	eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: kubeclientset.CoreV1().Events("")})
	recorder := eventBroadcaster.NewRecorder(scheme.Scheme, corev1.EventSource{Component: controllerAgentName})

	controller := &Controller{
		kubeclientset: kubeclientset,
		jobClientsets: jobClientsets,
		jobInformers:  jobInformers,
		workqueue:     workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "Queue"),
		unitMap:       map[string]*queue.QueueUnit{},
		recorder:      recorder,
	}

	klog.Info("Setting up event handlers")
	for jobType, informerInterface := range controller.jobInformers {
		switch jobType {
		case options.TFJob:
			informerFactory, ok := informerInterface.(externalversions.SharedInformerFactory)
			if !ok {
				klog.Fatalf("Failed to cast informer interface for %s", jobType)
			}
			informer := informerFactory.Kubeflow().V1().TFJobs().Informer()

			informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
				AddFunc:    controller.enqueueUnitForTFJob,
				DeleteFunc: controller.dequeueUnitForTFJob,
			})
			go informerFactory.Start(stopCh)
		}
	}

	return controller, nil
}

func (c *Controller) enqueueUnitForTFJob(obj interface{}) {
	tfjob, ok := obj.(*v1.TFJob)
	if !ok {
		return
	}

	if _, ok := tfjob.Annotations[annotationQueue]; !ok {
		return
	}

	unit := queue.QueueUnit{
		ObjectMeta: metav1.ObjectMeta{
			Name:      tfjob.Name,
			Namespace: tfjob.Namespace,
		},
		Spec: queue.Spec{
			PriorityClassName: "high",
			Queue:             "default",
		},
	}

	key := fmt.Sprintf("%s/%s/%s", options.TFJob, tfjob.Namespace, tfjob.Name)

	c.unitMap[key] = &unit
	c.workqueue.AddRateLimited(key)
}

func (c *Controller) dequeueUnitForTFJob(obj interface{}) {
	tfjob, ok := obj.(*v1.TFJob)
	if !ok {
		return
	}

	if _, ok := tfjob.Annotations[annotationQueue]; !ok {
		return
	}

	key := fmt.Sprintf("%s/%s/%s", options.TFJob, tfjob.Namespace, tfjob.Name)

	c.workqueue.Forget(key)
}

func (c *Controller) Run(threadiness int, stopCh <-chan struct{}) error {
	defer runtime.HandleCrash()
	defer c.workqueue.ShutDown()

	// Start the informer factories to begin populating the informer caches
	klog.Info("Starting Foo controller")

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
	obj, shutdown := c.workqueue.Get()

	if shutdown {
		return false
	}

	err := func(obj interface{}) error {
		defer c.workqueue.Done(obj)
		var key string
		var ok bool
		if key, ok = obj.(string); !ok {
			c.workqueue.Forget(obj)
			runtime.HandleError(fmt.Errorf("expected string in workqueue but got %#v", obj))
			return nil
		}
		if err := c.syncHandler(key); err != nil {
			return fmt.Errorf("error syncing '%s': %s", key, err.Error())
		}
		c.workqueue.Forget(obj)
		klog.Infof("Successfully synced '%s'", key)
		return nil
	}(obj)

	if err != nil {
		runtime.HandleError(err)
		return true
	}

	return true
}

func (c *Controller) syncHandler(key string) error {
	// TODO: @denkensk implement job release (remove annotation) step here
	klog.Infof("Processing key: %s", key)
	res := strings.Split(key, "/")
	if len(res) != 3 {
		return fmt.Errorf("parsing %s failed", key)
	}
	jobType := res[0]
	namespace := res[1]
	jobName := res[2]
	switch jobType {
	case options.TFJob:
		clientInterface, ok := c.jobClientsets[jobType]
		if !ok {
			klog.Fatalf("Failed to find client for %s", key)
		}
		client := clientInterface.(*tfjob.Clientset)
		job, err := client.KubeflowV1().TFJobs(namespace).Get(jobName, metav1.GetOptions{})
		if err != nil {
			if errors.IsNotFound(err) {
				c.workqueue.Forget(key)
				return nil
			}
			return err
		}
		// TODO: For prototyping, here we just sleep for 10 seconds
		// TODO: In real case, we should valid the resources requirement or other clearance checking for this job
		time.Sleep(10 * time.Second)
		delete(job.Annotations, annotationQueue)
		// update the tfjob
		_, err = client.KubeflowV1().TFJobs(namespace).Update(job)
		if err != nil {
			return err
		}
	}

	return nil
}
