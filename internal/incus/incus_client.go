package incus

import (
	"context"

	incus "github.com/lxc/incus/v6/client"
)

type Client struct {
}

type FooResponse struct {
	Project  string
	Location string
}

func (r *Client) GetContainerInfo(ctx context.Context, project string, container string) (FooResponse, error) {
	c, err := incus.ConnectIncusUnixWithContext(ctx, "", nil)
	if err != nil {
		return FooResponse{}, err
	}

	i, etag, err := c.UseProject(project).GetInstance(container)
	if err != nil {
		return FooResponse{}, err
	}

	// api.Response
	panic("")
}
