package app

import (
	"os"
	"time"

	"github.com/kube-queue/api/pkg/client/clientset/versioned"
	externalversions "github.com/kube-queue/api/pkg/client/informers/externalversions"

	"github.com/kube-queue/kube-queue/cmd/app/options"
	"github.com/kube-queue/kube-queue/pkg/controller"
	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"
	"k8s.io/sample-controller/pkg/signals"
)

const (
	apiVersion = "v1alpha1"
)

func Run(opt *options.ServerOption) error {
	klog.Infof("%+v", apiVersion)

	stopCh := signals.SetupSignalHandler()

	if len(os.Getenv("KUBECONFIG")) > 0 {
		opt.KubeConfig = os.Getenv("KUBECONFIG")
	}

	cfg, err := clientcmd.BuildConfigFromFlags("", opt.KubeConfig)
	if err != nil {
		klog.Fatalf("Error building kubeconfig: %s\n", err.Error())
	}

	kubeClient, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		klog.Fatalf("Error building kubernetes clientset: %s\n", err.Error())
	}

	kubeInformerFactory := kubeinformers.NewSharedInformerFactory(kubeClient, time.Second*30)
	restConfig, err := clientcmd.BuildConfigFromFlags("", opt.KubeConfig)
	if err != nil {
		return err
	}

	queueClient, err := versioned.NewForConfig(restConfig)
	if err != nil {
		return err
	}

	queueInformerFactory := externalversions.NewSharedInformerFactory(queueClient, 0)
	queueInformer := queueInformerFactory.Scheduling().V1alpha1().QueueUnits().Informer()

	qController, err := controller.NewController(kubeClient, opt.KubeConfig, kubeInformerFactory, queueClient, queueInformer)
	if err != nil {
		klog.Fatalln("Error building controller\n")
	}

	kubeInformerFactory.Start(stopCh)
	qController.Start()

	return nil
}
