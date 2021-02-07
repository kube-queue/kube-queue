package communicate

import (
	"fmt"

	"github.com/kube-queue/kube-queue/pkg/apis/queue/v1alpha1"
	"github.com/kube-queue/kube-queue/pkg/comm/extension/pb"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

type Client struct {
	pb.JobClient
	conn *grpc.ClientConn
	ctx  context.Context
}

func (c *Client) Close() error {
	return c.conn.Close()
}

func MakeExtensionClient(addr string) (ExtensionClientInterface, error) {
	conn, err := grpc.Dial(addr, grpc.WithInsecure())
	if err != nil {
		return nil, err
	}

	c := pb.NewJobClient(conn)

	return &Client{
		JobClient: c,
		conn:      conn,
		ctx:       context.Background(),
	}, nil
}

func (c *Client) ReleaseJob(qu *v1alpha1.QueueUnit) error {
	resp, err := c.Release(c.ctx, &pb.ReleaseRequest{
		Name:      qu.Name,
		Namespace: qu.Namespace,
	})
	if err != nil {
		return err
	}

	if resp.Message != "success" {
		return fmt.Errorf("failed to release job %s", qu.Name)
	}

	return nil
}
