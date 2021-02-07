package communicate

import (
	queue "github.com/kube-queue/kube-queue/pkg/apis/queue/v1alpha1"
	"github.com/kube-queue/kube-queue/pkg/comm/permission/pb"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

type Client struct {
	pb.PermissionClient
	conn *grpc.ClientConn
	ctx context.Context
}

func (c *Client) Close() error {
	return c.conn.Close()
}

func (c *Client) IfPermissionReceived(qu *queue.QueueUnit) (bool, error) {
	req := &pb.PermissionRequest{QueueUnit: qu}

	resp, err := c.DemandPermission(c.ctx, req)

	if err != nil {
		return false, err
	}

	return resp.Feedback == pb.PermissionFeedback_PROVE, nil
}

func MakePermissionClient(addr string) (PermissionClientInterface, error) {
	conn, err := grpc.Dial(addr, grpc.WithInsecure())
	if err != nil {
		return nil, err
	}

	c := pb.NewPermissionClient(conn)

	return &Client{
		PermissionClient: c,
		conn:             conn,
		ctx:              context.Background(),
	}, nil
}
