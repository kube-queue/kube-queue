package server

import (
	"os"

	pserver "github.com/kube-queue/kube-queue/third_party/permission-counter/server"

	"github.com/kube-queue/kube-queue/cmd/default-permission-counter/app/options"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"
)

func Run(opt *options.ServerOption) error {
	if len(os.Getenv("KUBECONFIG")) > 0 {
		opt.KubeConfig = os.Getenv("KUBECONFIG")
	}

	cfg, err := clientcmd.BuildConfigFromFlags("", opt.KubeConfig)
	if err != nil {
		klog.Fatalf("Error building kubeconfig: %s\n", err.Error())
	}

	grpcServer, err := pserver.MakeGRPCServerForPermissionCounter(cfg, opt.ListenTo)
	if err != nil {
		return err
	}

	return grpcServer.Serve()
}
