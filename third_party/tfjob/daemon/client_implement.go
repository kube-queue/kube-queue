package daemon

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strings"

	queue "github.com/kube-queue/kube-queue/pkg/apis/queue/v1alpha1"
	communicate "github.com/kube-queue/kube-queue/pkg/comm/queue"
	"golang.org/x/net/context"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog"
)

const TFJob = "tfjob"

type Client struct {
	core       http.Client
	ctx        context.Context
	addrPrefix string
}

func (c *Client) Close() error {
	return c.Close()
}

func MakeQueueClient(addr string) (communicate.QueueClientInterface, error) {
	fileAddr := strings.TrimPrefix(addr, "unix://")
	c := http.Client{
		Transport: &http.Transport{
			DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
				return net.Dial("unix", fileAddr)
			},
		},
	}

	return &Client{
		core:       c,
		addrPrefix: "http://localhost",
		ctx:        context.Background(),
	}, nil
}

func (c *Client) AddFunc(obj interface{}) {
	metaObj, resource, err := extractFromUnstructured(obj)
	if err != nil {
		return
	}

	jph := queue.JobDequeued
	annotation := metaObj.GetAnnotations()
	if annotation != nil {
		if _, ok := annotation[queue.AnnotationQueue]; ok {
			jph = queue.JobEnqueued
		}
	}

	qu := queue.QueueUnit{
		ObjectMeta: metav1.ObjectMeta{
			Name:      metaObj.GetName(),
			Namespace: metaObj.GetNamespace(),
			UID:       metaObj.GetUID(),
		},
		Spec: queue.Spec{
			JobType:  TFJob,
			Resource: resource,
		},
		Status: queue.Status{
			Phase: jph,
		},
	}

	data, err := json.Marshal(qu)
	if err != nil {
		klog.Warningf("failed to convert %v to json\n", qu)
		return
	}
	klog.Infof("sending Add request to queue-controller for: %s", string(data))

	_, err = c.core.Post(
		fmt.Sprintf("%s/", c.addrPrefix), http.DetectContentType(data), strings.NewReader(string(data)))
	if err != nil {
		klog.Warningf("failed to send request: %v", err)
	}
	return
}

func (c *Client) DeleteFunc(obj interface{}) {
	metaObj, _, err := extractFromUnstructured(obj)
	if err != nil {
		return
	}

	qu := &queue.QueueUnit{
		ObjectMeta: metav1.ObjectMeta{
			Name:      metaObj.GetName(),
			Namespace: metaObj.GetNamespace(),
			UID:       metaObj.GetUID(),
		},
		Spec: queue.Spec{
			JobType: TFJob,
		},
		Status: queue.Status{
			Phase: queue.JobDequeued,
		},
	}

	data, err := json.Marshal(qu)
	if err != nil {
		return
	}
	deleteRequest, err := http.NewRequest(http.MethodDelete, fmt.Sprintf("%s/", c.addrPrefix), strings.NewReader(string(data)))
	if err != nil {
		return
	}
	deleteRequest.Header.Set("Content-type", "application/json")

	_, err = c.core.Do(deleteRequest)
	return
}
