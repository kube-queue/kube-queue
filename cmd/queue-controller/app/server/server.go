package app

import (
	"io/ioutil"
	"os"

	"github.com/kube-queue/kube-queue/cmd/queue-controller/app/options"
	extension "github.com/kube-queue/kube-queue/pkg/comm/extension"
	permission "github.com/kube-queue/kube-queue/pkg/comm/permission"
	communicate "github.com/kube-queue/kube-queue/pkg/comm/queue"
	"github.com/kube-queue/kube-queue/pkg/controller"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
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

	// Setup the Permission Counter Client
	pclient, err := permission.MakePermissionClient(opt.PermissionCounterAddr)
	if err != nil {
		return err
	}

	// Create Extension Client (for release job)
	data, err := ioutil.ReadFile(opt.ExtensionConfig)
	extConfig := &options.ExtensionConfig{TypeAddr: map[string]string{}}
	err = yaml.Unmarshal(data, extConfig)
	if err != nil {
		return err
	}
	extClients := map[string]extension.ExtensionClientInterface{}
	for jobType, addr := range extConfig.TypeAddr {
		ec, err := extension.MakeExtensionClient(addr)
		if err != nil {
			return err
		}
		extClients[jobType] = ec
	}

	qController, err := controller.NewController(kubeClient, pclient, extClients)
	if err != nil {
		klog.Fatalln("Error building controller\n")
	}

	// Setup the GRPC Server
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
