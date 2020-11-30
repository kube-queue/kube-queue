package app

import (
	"os"

	"github.com/kube-queue/kube-queue/cmd/app/options"
	"github.com/kube-queue/kube-queue/pkg/controller"
	tfjob "github.com/kubeflow/tf-operator/pkg/client/clientset/versioned"
	"github.com/kubeflow/tf-operator/pkg/client/informers/externalversions"
	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"
	"k8s.io/sample-controller/pkg/signals"
)

const (
	apiVersion = "v1alpha1"
)

const (
	QueueNamespace = "QUEUE_NAMESPACE"
	KubeConfigPath = "KUBECONFIG"
)

func Run(opt *options.ServerOption) error {
	// Set namespace
	namespace := os.Getenv(QueueNamespace)
	if len(namespace) == 0 {
		log.Infof("%s not set, use default\n", QueueNamespace)
		namespace = "default"
	}
	if opt.Namespace == corev1.NamespaceAll {
		log.Info("Using cluster scoped operator")
	} else {
		log.Infof("Scoping operator to namespace %s", opt.Namespace)
	}

	log.Infof("%+v", apiVersion)

	stopCh := signals.SetupSignalHandler()

	if len(os.Getenv(KubeConfigPath)) > 0 {
		opt.Kubeconfig = os.Getenv(KubeConfigPath)
	}

	cfg, err := clientcmd.BuildConfigFromFlags("", opt.Kubeconfig)
	if err != nil {
		klog.Fatalf("Error building kubeconfig: %s\n", err.Error())
	}

	kubeClient, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		klog.Fatalf("Error building kubernetes clientset: %s\n", err.Error())
	}

	jobClients := make(map[string]interface{})
	jobInformers := make(map[string]interface{})
	for _, jobType := range opt.EnabledJobs {
		switch jobType {
		case options.TFJob:
			client, err := tfjob.NewForConfig(cfg)
			if err != nil {
				klog.Fatalf("Error building %s clientset: %s\n", options.TFJob, err.Error())
			}
			jobClients[options.TFJob] = client
			informer := externalversions.NewSharedInformerFactory(client, opt.ResyncPeriod)
			jobInformers[options.TFJob] = informer

		default:
			klog.Fatalf("Job %s not supported\n", jobType)
		}
	}

	qController, err := controller.NewController(kubeClient, jobClients, jobInformers, stopCh)
	if err != nil {
		klog.Fatalln("Error building controller\n")
	}

	if err = qController.Run(2, stopCh); err != nil {
		klog.Fatalf("Error running controller: %s", err.Error())
	}

	return nil
}
