package communicate

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strings"

	"github.com/kube-queue/kube-queue/pkg/apis/queue/v1alpha1"
	"golang.org/x/net/context"
)

type Client struct {
	core       http.Client
	ctx        context.Context
	addrPrefix string
}

func (c *Client) Close() error {
	return c.Close()
}

func MakeExtensionClient(addr string) (ExtensionClientInterface, error) {
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
		ctx:        context.Background(),
		addrPrefix: "http://localhost",
	}, nil
}

func (c *Client) DequeueJob(qu *v1alpha1.QueueUnit) error {
	data, err := json.Marshal(qu)
	if err != nil {
		return err
	}

	releaseRequest, err := http.NewRequest(
		http.MethodDelete, fmt.Sprintf("%s/", c.addrPrefix), strings.NewReader(string(data)))
	if err != nil {
		return err
	}
	releaseRequest.Header.Set("Content-type", "application/json")

	_, err = c.core.Do(releaseRequest)
	return err
}
