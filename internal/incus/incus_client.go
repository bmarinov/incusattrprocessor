package incus

import (
	"context"
	"fmt"
	"strings"

	incusclient "github.com/lxc/incus/v6/client"
)

// Client looks up Incus instance metadata via the local Unix socket.
type Client struct {
	server incusclient.InstanceServer
	conn   connectFunc
}

type connectFunc func(ctx context.Context) (incusclient.InstanceServer, error)

type InstanceInfo struct {
	Name     string
	Project  string
	Location string
	// TODO: check if not already present in ebpf profile attrs
	Architecture string
	// TODO: cpu limits
}

func New(socketPath string) *Client {
	return &Client{
		server: nil,
		conn: func(ctx context.Context) (incusclient.InstanceServer, error) {

			conn, err := incusclient.ConnectIncusUnixWithContext(ctx, socketPath, nil)
			if err != nil {
				return nil, fmt.Errorf("connecting to incus daemon: %w", err)
			}
			return conn, nil
		},
	}
}

// GetInstance returns the cluster member (location) hosting the instance.
func (c *Client) GetInstance(ctx context.Context, project, name string) (InstanceInfo, error) {
	inst, _, err := c.server.UseProject(project).GetInstance(name)
	if err != nil {
		return InstanceInfo{}, fmt.Errorf("incus get instance %s/%s: %w", project, name, err)
	}

	// limits.cpu.allowance || limits.cpu
	// for k, cfg := range inst.ExpandedConfig {
	// 	slog.Info("cfg", k, cfg)
	// }

	return InstanceInfo{
		Name:         name,
		Project:      project,
		Location:     inst.Location,
		Architecture: inst.Architecture,
	}, nil
}

// SplitLabel splits an LXC cgroup label into a project and instance name.
// For instances with no project prefix, "default" is returned.
func SplitLabel(label string) (project, name string) {
	project, name, ok := strings.Cut(label, "_")
	if !ok {
		return "default", label
	}
	return project, name
}

func (c *Client) Start(ctx context.Context) error {
	conn, err := c.conn(ctx)
	if err != nil {
		return fmt.Errorf("connecting to incus daemon: %w", err)
	}

	c.server = conn

	return nil
}
