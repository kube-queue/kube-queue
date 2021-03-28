package main

import (
	"flag"
	"os"

	"github.com/kube-queue/kube-queue/cmd/example-bundles/tf-job/options"
	"github.com/kube-queue/kube-queue/third_party/tfjob/daemon"
	"github.com/kube-queue/kube-queue/third_party/tfjob/server"
	tfjobClient "github.com/kubeflow/tf-operator/pkg/client/clientset/versioned"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog"
	"k8s.io/sample-controller/pkg/signals"
)

const (
	TFJobNamespace = "TFJOB_NAMESPACE"
	KubeConfigPath = "KUBECONFIG"
)

func main() {
	opt := options.NewServerOption()
	opt.AddFlags(flag.CommandLine)

	flag.Parse()

	// Set Namespace
	namespace := os.Getenv(TFJobNamespace)
	if len(namespace) > 0 {
		opt.Namespace = namespace
	}

	stopCh := signals.SetupSignalHandler()

	if len(os.Getenv(KubeConfigPath)) > 0 {
		opt.Kubeconfig = os.Getenv(KubeConfigPath)
	}
	cfg, err := clientcmd.BuildConfigFromFlags("", opt.Kubeconfig)
	if err != nil {
		klog.Fatalf("Error building kubeconfig: %s\n", err.Error())
	}

	d, err := daemon.MakeDaemon(cfg, opt.QueueAddr, opt.Namespace)
	if err != nil {
		klog.Fatalf("Error making daemon for tfjob: %s\n", err.Error())
	}

	d.Run(stopCh)

	tfci, err := tfjobClient.NewForConfig(cfg)
	if err != nil {
		klog.Fatalln(err)
	}
	extensionServer := server.MakeExtensionServer(tfci)

	if err = server.StartServer(extensionServer, opt.ListenTo); err != nil {
		klog.Fatalln(err)
	}
}
