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

package controller

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog/v2"

	"github.com/kube-queue/api/pkg/client/clientset/versioned"
	"github.com/kube-queue/kube-queue/pkg/framework"
	"github.com/kube-queue/kube-queue/pkg/framework/plugins"
	"github.com/kube-queue/kube-queue/pkg/framework/runtime"
	"github.com/kube-queue/kube-queue/pkg/queue"
	"github.com/kube-queue/kube-queue/pkg/queue/multischedulingqueue"
	"github.com/kube-queue/kube-queue/pkg/scheduler"
	"github.com/kube-queue/kube-queue/pkg/utils"
)

type Controller struct {
	// recorder is an event recorder for recording Event resources to the
	// Kubernetes API.
	recorder             record.EventRecorder
	multiSchedulingQueue queue.MultiSchedulingQueue
	fw                   framework.Framework
	scheduler            *scheduler.Scheduler
	queueUnitInformer    cache.SharedIndexInformer
	queueUnitClient      *versioned.Clientset
}

func NewController(
	kubeClient kubernetes.Interface,
	kubeConfigPath string,
	informersFactory informers.SharedInformerFactory,
	queueUnitClient *versioned.Clientset,
	queueInformer cache.SharedIndexInformer,
	stopCh <-chan struct{},
	podInitialBackoffSeconds int,
	podMaxBackoffSeconds int) (*Controller, error) {

	// Create event broadcaster
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(klog.Infof)
	eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: kubeClient.CoreV1().Events("")})

	schemeModified := scheme.Scheme
	recorder := eventBroadcaster.NewRecorder(schemeModified, corev1.EventSource{Component: utils.ControllerAgentName})

	r := plugins.NewInTreeRegistry()
	fw, err := runtime.NewFramework(r, kubeConfigPath, informersFactory, queueUnitClient)
	if err != nil {
		klog.Fatalf("new framework failed %v", err)
	}

	multiSchedulingQueue, err := multischedulingqueue.NewMultiSchedulingQueue(fw, podInitialBackoffSeconds, podMaxBackoffSeconds)
	if err != nil {
		klog.Fatalf("init multi scheduling queue failed %s", err)
	}

	controller := &Controller{
		recorder:             recorder,
		fw:                   fw,
		multiSchedulingQueue: multiSchedulingQueue,
		queueUnitClient:      queueUnitClient,
		queueUnitInformer:    queueInformer,
	}
	controller.addAllEventHandlers(queueInformer)
	go controller.queueUnitInformer.Run(stopCh)

	controller.scheduler, err = scheduler.NewScheduler(multiSchedulingQueue, fw, queueUnitClient)
	if err != nil {
		klog.Fatalf("init scheduler failed %s", err)
	}

	return controller, nil
}

func (c *Controller) Start(ctx context.Context) {
	c.multiSchedulingQueue.Run()
	c.scheduler.Start(ctx)
	c.multiSchedulingQueue.Close()
}
