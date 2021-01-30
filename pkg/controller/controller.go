package controller

import (
	"fmt"
	"strings"
	"time"

	"github.com/kube-queue/kube-queue/cmd/app/options"
	queue "github.com/kube-queue/kube-queue/pkg/apis/queue/v1alpha1"
	internalcache "github.com/kube-queue/kube-queue/pkg/cache"
	v1 "github.com/kubeflow/tf-operator/pkg/apis/tensorflow/v1"
	tfjobClientset "github.com/kubeflow/tf-operator/pkg/client/clientset/versioned"
	tfjobInformerv1 "github.com/kubeflow/tf-operator/pkg/client/informers/externalversions/tensorflow/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
	// kubeclientset is a standard kubernetes clientset
	kubeclientset kubernetes.Interface
	// jobClientset is a map of job type: clientset for enabled job type
	jobClientsets map[string]interface{}
	// jobInformers is a map of job type: informer for enabled job type
	jobInformers map[string]interface{}

	// workqueue is the where queue units are stored
	WorkQueue workqueue.RateLimitingInterface

	// namespacedJobSets maps key (namespace) to the Queue Unit
	// it performs as an internal cache for resource reservation
	namespacedJobSet internalcache.JobCacheInterface

	// recorder is an event recorder for recording Event resources to the
	// Kubernetes API.
	recorder record.EventRecorder
}

func NewController(kubeclientset kubernetes.Interface,
	jobClientsets map[string]interface{},
	jobInformers map[string]interface{},
	namespaces []string,
	stopCh <-chan struct{},
) (*Controller, error) {
	// Create event broadcaster
	// Add sample-controller types to the default Kubernetes Scheme so Events can be
	// logged for sample-controller types.
	klog.V(4).Info("Creating event broadcaster")
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(klog.Infof)
	eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: kubeclientset.CoreV1().Events("")})

	schemeModified := scheme.Scheme
	err := v1.AddToScheme(schemeModified)
	if err != nil {
		return nil, err
	}
	recorder := eventBroadcaster.NewRecorder(schemeModified, corev1.EventSource{Component: controllerAgentName})

	//
	nsRMap := map[string]corev1.ResourceList{}
	for _, ns := range namespaces {
		nsRMap[ns] = corev1.ResourceList{}
	}

	controller := &Controller{
		kubeclientset: kubeclientset,
		jobClientsets: jobClientsets,
		jobInformers:  jobInformers,
		WorkQueue: workqueue.NewNamedRateLimitingQueue(
			workqueue.NewItemFastSlowRateLimiter(2*time.Second, 1*time.Minute, 50), "priority"),
		namespacedJobSet: internalcache.MakeInternalCache(),
		recorder:         recorder,
	}

	klog.Info("Setting up event handlers")
	for jobType, informerInterface := range controller.jobInformers {
		switch jobType {
		case options.TFJob:
			informer, ok := informerInterface.(*tfjobInformerv1.TFJobInformer)
			if !ok {
				return nil, fmt.Errorf("failed to convert interface to ")
			}

			(*informer).Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
				AddFunc:    controller.enqueueItem,
				DeleteFunc: controller.dequeueItem,
			})
			go (*informer).Informer().Run(stopCh)
		}
	}

	return controller, nil
}

func (c *Controller) enqueueItem(obj interface{}) {
	metaObj, resource, err := extractFromUnstructured(obj)
	if err != nil {
		return
	}

	jph := queue.JobDequeued
	annotation := metaObj.GetAnnotations()
	if annotation != nil {
		if _, ok := annotation[annotationQueue]; ok {
			jph = queue.JobEnqueued
		}
	}

	qu := queue.QueueUnit{
		ObjectMeta: metav1.ObjectMeta{
			Name:      metaObj.GetName(),
			Namespace: metaObj.GetNamespace(),
			UID:       metaObj.GetUID(),
		},
		Spec: queue.Spec{
			JobType:  options.TFJob,
			Resource: resource,
		},
		Status: queue.Status{
			Phase: jph,
		},
	}

	c.namespacedJobSet.AddOrUpdate(qu)

	key := qu.Serialize()

	if jph == queue.JobEnqueued {
		c.WorkQueue.AddRateLimited(key)
	}
}

func (c *Controller) dequeueItem(obj interface{}) {
	metaObj, _, err := extractFromUnstructured(obj)
	if err != nil {
		return
	}

	c.namespacedJobSet.Remove(metaObj.GetNamespace(), metaObj.GetUID())

	key := queue.MakeSimpleQueueUnit(metaObj.GetName(), metaObj.GetNamespace(), options.TFJob).Serialize()

	c.WorkQueue.Forget(key)
}

func (c *Controller) updateItem(oldObj, newObj interface{}) {
	c.enqueueItem(newObj)
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

	metaJob, err := queue.Deserialize(key)
	if err != nil {
		return false, err
	}
	jobType := metaJob.Spec.JobType
	namespace := metaJob.Namespace
	jobName := metaJob.Name

	switch jobType {
	case options.TFJob:
		clientInterface, ok := c.jobClientsets[jobType]
		if !ok {
			klog.Fatalf("Failed to find client for %s", key)
		}
		client := clientInterface.(*tfjobClientset.Clientset)
		j, err := client.KubeflowV1().TFJobs(namespace).Get(jobName, metav1.GetOptions{})
		if err != nil {
			return false, err
		}
		if j.Annotations == nil {
			return true, nil
		}
		if _, annotationExist := j.Annotations[annotationQueue]; !annotationExist {
			return true, nil
		}

		resourceRequest := CalResourceRequestForTFJob(j)
		reservedQuota := c.namespacedJobSet.ReservedResource(namespace)

		// If there are multiple resource quota within the namespace, we accumulate all of them
		// TODO: Replace List from kubeclientset to Lister
		totalQuota := corev1.ResourceList{}
		rqs, err := c.kubeclientset.CoreV1().ResourceQuotas(namespace).List(metav1.ListOptions{})
		if err != nil {
			return false, err
		}
		for _, rq := range rqs.Items {
			for rType, rQuantity := range rq.Spec.Hard {
				if strings.HasPrefix(rType.String(), "requests.") {
					storedRType := strings.ReplaceAll(rType.String(), "requests.", "")
					if accumulated, exist := totalQuota[corev1.ResourceName(storedRType)]; exist {
						rQuantity.Add(accumulated)
					}
					totalQuota[corev1.ResourceName(storedRType)] = rQuantity
				}

			}
		}

		if EnoughResource(resourceRequest, reservedQuota, totalQuota) {
			delete(j.Annotations, annotationQueue)
			// update the tfjob
			_, err = client.KubeflowV1().TFJobs(namespace).Update(j)
			if err != nil {
				return false, err
			}
			// update reserved
			c.namespacedJobSet.UpdatePhase(namespace, j.UID, queue.JobDequeued)
			klog.Infof("job %s cleared", key)
		} else {
			// If there is not enough resource left for the job, return non-error and not to forget this job
			return false, fmt.Errorf("not enough resource left for job %s", key)
		}
	}

	return true, nil
}
