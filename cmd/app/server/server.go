/*
 Copyright 2021 The Kube-Queue Authors.

 Licensed under the Apache License, Version 2.0 (the "License");
 you may not use this file except in compliance with the License.
 You may obtain a copy of the License at

     http://www.apache.org/licenses/LICENSE-2.0

 Unless required by applicable law or agreed to in writing, software
 distributed under the License is distributed on an "AS IS" BASIS,
 WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 See the License for the specific language governing permissions and
 limitations under the License.
*/

package app

import (
	"context"
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
)

const (
	apiVersion = "v1alpha1"
)

func Run(opt *options.ServerOption) error {
	klog.Infof("%+v", apiVersion)

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
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	qController, err := controller.NewController(kubeClient, opt.KubeConfig, kubeInformerFactory, queueClient, queueInformer, ctx.Done())
	if err != nil {
		klog.Fatalln("Error building controller\n")
	}

	kubeInformerFactory.Start(ctx.Done())
	qController.Start(ctx)

	return nil
}
