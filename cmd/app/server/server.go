package app

import (
	"io/ioutil"
	"os"
	"time"

	communicate "github.com/kube-queue/kube-queue/pkg/comm/queue"

	"github.com/kube-queue/kube-queue/cmd/app/options"
	extension "github.com/kube-queue/kube-queue/pkg/comm/extension"
	"github.com/kube-queue/kube-queue/pkg/controller"
	"github.com/kube-queue/kube-queue/pkg/permission"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
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
	log.Infof("%+v", apiVersion)

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
	quotaList := kubeInformerFactory.Core().V1().ResourceQuotas().Lister()

	// Setup the Permission Counter Client
	pc := permission.MakeResourcePermissionCounter(quotaList)

	// Create Extension Client (for release job)
	data, err := ioutil.ReadFile(opt.ExtensionConfig)
	typeAddr := make(map[string]string)
	err = yaml.Unmarshal(data, &typeAddr)
	if err != nil {
		return err
	}
	extClients := map[string]extension.ExtensionClientInterface{}
	for jobType, addr := range typeAddr {
		ec, err := extension.MakeExtensionClient(addr)
		if err != nil {
			return err
		}
		extClients[jobType] = ec
	}

	qController, err := controller.NewController(kubeClient, pc, extClients)
	if err != nil {
		klog.Fatalln("Error building controller\n")
	}

	kubeInformerFactory.Start(stopCh)

	// Setup the Server
	server := communicate.MakeQueueServer(qController)

	go func() {
		if serverErr := communicate.StartServer(server, opt.ListenTo); serverErr != nil {
			klog.Fatalln("server stopped!")
		}
	}()

	if err = qController.Run(2, stopCh); err != nil {
		klog.Fatalf("Error running controller: %s", err.Error())
	}

	return nil
}
