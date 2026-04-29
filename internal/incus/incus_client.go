package incus

import (
	"context"
)

type Client struct {
}

type FooResponse struct {
	Project  string
	Location string
}

func (c *Client) GetContainerInfo(ctx context.Context, container string) (FooResponse, error) {
	// api.Response
	panic("")
}
