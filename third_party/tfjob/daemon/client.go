package daemon

import (
	"github.com/kube-queue/kube-queue/cmd/app/options"
	queue "github.com/kube-queue/kube-queue/pkg/apis/queue/v1alpha1"
	communicate "github.com/kube-queue/kube-queue/pkg/comm/controller"
	"github.com/kube-queue/kube-queue/pkg/comm/controller/pb"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Client struct {
	pb.QueueClient
	conn *grpc.ClientConn
	ctx context.Context
}

func MakeClient(addr string) (communicate.ControllerClientInterface, error) {
	conn, err := grpc.Dial(addr, grpc.WithInsecure())
	if err != nil {
		return nil, err
	}

	c := pb.NewQueueClient(conn)

	return &Client{
		QueueClient: c,
		conn:        conn,
		ctx:         context.Background(),
	}, nil
}

func (c *Client) Close() error {
	return c.conn.Close()
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
			JobType:  options.TFJob,
			Resource: resource,
		},
		Status: queue.Status{
			Phase: jph,
		},
	}

	_, _ = c.AddOrUpdate(c.ctx, &pb.AddRequest{QueueUnit: &qu})
}

func (c *Client) DeleteFunc(obj interface{}) {
	metaObj, _, err := extractFromUnstructured(obj)
	if err != nil {
		return
	}

	name := metaObj.GetName()
	namespace := metaObj.GetNamespace()
	uid := string(metaObj.GetUID())
	jobType := "tf-job"

	_, _ = c.Delete(c.ctx, &pb.DeleteRequest{
		Name:      name,
		Namespace: namespace,
		Uid:       uid,
		Type:      jobType,
	})
}