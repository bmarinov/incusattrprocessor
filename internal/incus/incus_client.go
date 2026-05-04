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
}

type InstanceInfo struct {
	Location string
	// TODO: check if not already present in ebpf profile attrs
	Architecture string
	// TODO: cpu limits
}

// New returns an API client with a live connection.
func New(conn incusclient.InstanceServer) *Client {
	return &Client{server: conn}
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
