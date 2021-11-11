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

	cfg.QPS = float32(opt.QPS)
	cfg.Burst = opt.Burst
	kubeClient, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		klog.Fatalf("Error building kubernetes clientset: %s\n", err.Error())
	}

	kubeInformerFactory := kubeinformers.NewSharedInformerFactory(kubeClient, time.Second*30)
	restConfig, err := clientcmd.BuildConfigFromFlags("", opt.KubeConfig)
	if err != nil {
		return err
	}
	restConfig.QPS = float32(opt.QPS)
	restConfig.Burst = opt.Burst
	queueUnitClient, err := versioned.NewForConfig(restConfig)
	if err != nil {
		return err
	}

	queueUnitInformerFactory := externalversions.NewSharedInformerFactory(queueUnitClient, 0)
	queueUnitInformer := queueUnitInformerFactory.Scheduling().V1alpha1().QueueUnits().Informer()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	controller, err := controller.NewController(kubeClient, opt.KubeConfig, kubeInformerFactory, queueUnitClient, queueUnitInformer, ctx.Done(), opt.PodInitialBackoffSeconds, opt.PodMaxBackoffSeconds)
	if err != nil {
		klog.Fatalln("Error building controller\n")
	}

	klog.Infof("Start successfully")
	kubeInformerFactory.Start(ctx.Done())
	controller.Start(ctx)

	return nil
}
