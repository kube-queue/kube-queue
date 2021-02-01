package communicate

import (
	"fmt"
	"github.com/kube-queue/kube-queue/pkg/apis/queue/v1alpha1"
	"github.com/kube-queue/kube-queue/pkg/comm/bundle/pb"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

type Client struct {
	pb.QueueClient
	conn *grpc.ClientConn
	ctx context.Context
}

func (c *Client) Close() error {
	return c.conn.Close()
}

func MakeClient(addr string) (BundleClientInterface, error) {
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

func (c *Client) ReleaseJob(key string) error {
	qu, err := v1alpha1.Deserialize(key)
	if err != nil {
		return err
	}

	resp, err := c.Release(c.ctx, &pb.ReleaseRequest{
		Name:      qu.Name,
		Namespace: qu.Namespace,
	})
	if err != nil {
		return err
	}

	if resp.Message != "success" {
		return fmt.Errorf("failed to release job %s", key)
	}

	return nil
}