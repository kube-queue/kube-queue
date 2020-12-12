package controller

import (
	"fmt"
	"strings"
	"time"

	"github.com/kube-queue/kube-queue/cmd/app/options"
	queue "github.com/kube-queue/kube-queue/pkg/apis/queue/v1alpha1"
	queuejob "github.com/kube-queue/kube-queue/pkg/job"
	v1 "github.com/kubeflow/tf-operator/pkg/apis/tensorflow/v1"
	tfjobClientset "github.com/kubeflow/tf-operator/pkg/client/clientset/versioned"
	tfjobInformerv1 "github.com/kubeflow/tf-operator/pkg/client/informers/externalversions/tensorflow/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	metav1unstructured "k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
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
	WorkQueue workqueue.RateLimitingInterface

	// namespacedJobSets maps key (namespace) to the Queue Unit
	// it performs as an internal cache for resource reservation
	namespacedJobSet map[string][]queue.QueueUnit

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
		namespacedJobSet: map[string][]queue.QueueUnit{},
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

func (c *Controller) register(obj metav1.Object, res corev1.ResourceList, phase queue.JobPhase) {
	namespace := obj.GetNamespace()
	qu := queue.QueueUnit{
		ObjectMeta: metav1.ObjectMeta{
			Name:      obj.GetName(),
			Namespace: obj.GetNamespace(),
			// Because we only use QueueUnit internally, we set the UID of the QueueUnit identical to the Job
			UID: obj.GetUID(),
		},
		Spec: queue.Spec{
			PriorityClassName: "high",
			Queue:             "default",
			Resource:          res,
		},
		Status: queue.Status{
			Phase: phase,
		},
	}

	q, _ := c.namespacedJobSet[namespace]
	if q == nil {
		q = append(q, qu)
	} else {
		registered := false
		for idx, item := range q {
			if item.UID == qu.UID {
				q[idx] = qu
				registered = true
			}
		}
		if !registered {
			q = append(q, qu)
		}
	}
	c.namespacedJobSet[namespace] = q
}

func (c *Controller) unregister(obj metav1.Object) {
	uid := obj.GetUID()
	namespace := obj.GetNamespace()

	q, nsExist := c.namespacedJobSet[namespace]
	if !nsExist {
		return
	}

	if q != nil {
		for idx, qu := range q {
			if qu.UID == uid {
				q = append(q[:idx], q[idx+1:]...)
			}
		}
		c.namespacedJobSet[namespace] = q
	}
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

	c.register(metaObj, resource, jph)

	key := queuejob.GenericJob{
		Name:      metaObj.GetName(),
		Namespace: metaObj.GetNamespace(),
		Kind:      options.TFJob,
	}.String()

	if jph == queue.JobEnqueued {
		c.WorkQueue.AddRateLimited(key)
	}
}

func (c *Controller) dequeueItem(obj interface{}) {
	metaObj, _, err := extractFromUnstructured(obj)
	if err != nil {
		return
	}

	c.unregister(metaObj)

	key := queuejob.GenericJob{
		Name:      metaObj.GetName(),
		Namespace: metaObj.GetNamespace(),
		Kind:      options.TFJob,
	}.String()

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

	metaJob, err := queuejob.ConvertToGenericJob(key)
	if err != nil {
		return false, err
	}
	jobType := metaJob.Kind
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
		reservedQuota := c.getReserved(namespace)

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
			c.updateJobPhase(namespace, j.UID, queue.JobDequeued)
			klog.Infof("job %s cleared", key)
		} else {
			// If there is not enough resource left for the job, return non-error and not to forget this job
			return false, fmt.Errorf("not enough resource left for job %s", key)
		}
	}

	return true, nil
}

func (c *Controller) getReserved(namespace string) corev1.ResourceList {
	q, exist := c.namespacedJobSet[namespace]
	if !exist {
		return nil
	}

	reserved := corev1.ResourceList{}
	for _, qu := range q {
		if qu.Status.Phase == queue.JobDequeued {
			for resName, resVal := range qu.Spec.Resource {
				newVal := resVal.DeepCopy()
				oldVal, resExist := reserved[resName]
				if resExist {
					newVal.Add(oldVal)
				}
				reserved[resName] = newVal
			}
		}
	}

	return reserved
}

func (c *Controller) updateJobPhase(namespace string, uid types.UID, newPhase queue.JobPhase) {
	q, exist := c.namespacedJobSet[namespace]
	if !exist {
		return
	}

	for idx, qu := range q {
		if qu.UID == uid {
			q[idx].Status.Phase = newPhase
		}
	}
}

func CalResourceRequestForTFJob(j *v1.TFJob) corev1.ResourceList {
	resource := corev1.ResourceList{}
	for _, spec := range j.Spec.TFReplicaSpecs {
		replicas := int(*spec.Replicas)
		for _, c := range spec.Template.Spec.Containers {
			if c.Resources.Requests != nil {
				for resourceType, resourceQuantity := range c.Resources.Requests {
					for i := 0; i < replicas-1; i++ {
						resourceQuantity.Add(resourceQuantity)
					}
					oldQuantity, ok := resource[resourceType]
					if ok {
						resourceQuantity.Add(oldQuantity)
					}
					resource[resourceType] = resourceQuantity
				}
			}
		}
	}
	return resource
}

func EnoughResource(jobResource corev1.ResourceList, reserved corev1.ResourceList, quota corev1.ResourceList) bool {
	for rType, rQuantity := range jobResource {
		// If resource type not defined, then prohibit it
		quotaQuantity, exist := quota[rType]
		if !exist {
			return false
		}

		quotaCopy := quotaQuantity.DeepCopy()
		// calculate remaining quantity
		if reservedQuantity, exist := reserved[rType]; exist {
			quotaCopy.Sub(reservedQuantity)
		}

		// finally calculate if the remaining value is enough for the job
		quotaCopy.Sub(rQuantity)
		if quotaCopy.Sign() == -1 {
			return false
		}
	}
	return true
}

func extractFromUnstructured(obj interface{}) (metav1.Object, corev1.ResourceList, error) {
	un, ok := obj.(*metav1unstructured.Unstructured)
	if !ok {
		return nil, nil, fmt.Errorf("cannot convert object to Unstructured")
	}

	switch un.GetKind() {
	case v1.Kind:
		var tfjob v1.TFJob
		err := runtime.DefaultUnstructuredConverter.FromUnstructured(un.Object, &tfjob)
		if err != nil {
			return nil, nil, fmt.Errorf("cannot convert to tfjob")
		}
		return &tfjob, CalResourceRequestForTFJob(&tfjob), nil
	default:
		return nil, nil, fmt.Errorf("type %s is not supported", un.GetKind())
	}
}
