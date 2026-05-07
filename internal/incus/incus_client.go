package incus

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"strings"
	"syscall"
	"time"

	incusclient "github.com/lxc/incus/v6/client"
	"github.com/lxc/incus/v6/shared/api"
)

// Client looks up Incus instance metadata via the local Unix socket.
type Client struct {
	// server connection with the Incus daemon
	server  incusclient.InstanceServer
	connect connectFunc
	// rootCtx used on reconnect
	rootCtx context.Context
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
		connect: func(ctx context.Context) (incusclient.InstanceServer, error) {
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
	inst, err := withReconnect(c, ctx, func() (*api.Instance, error) {
		inst, _, err := c.server.UseProject(project).GetInstance(name)
		return inst, err
	})

	if err != nil {
		return InstanceInfo{}, fmt.Errorf("incus get instance %s/%s: %w", project, name, err)
	}

	// limits.cpu.allowance || limits.cpu
	// for k, cfg := range inst.ExpandedConfig {
	// 	slog.Info("cfg", k, cfg)
	// }

	return toInfo(inst), nil
}

func (c *Client) GetAllInstances(ctx context.Context) ([]InstanceInfo, error) {
	instances, err := withReconnect(c, ctx, func() ([]api.Instance, error) {
		return c.server.GetInstancesAllProjects(api.InstanceTypeAny)
	})

	if err != nil {
		return nil, fmt.Errorf("fetching all instances: %w", err)
	}
	var result []InstanceInfo
	for _, inst := range instances {
		result = append(result, toInfo(&inst))
	}

	return result, nil
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

func isUnreachable(err error) bool {
	return errors.Is(err, syscall.ECONNREFUSED) || errors.Is(err, fs.ErrNotExist)
}

func (c *Client) Start(ctx context.Context) error {
	c.rootCtx = ctx

	for {
		srvConn, err := retry(ctx,
			func() (result incusclient.InstanceServer, err error) {
				return c.connect(ctx)
			}, isUnreachable)

		if err == nil {
			c.server = srvConn
			return nil
		}
		if !isUnreachable(err) {
			return fmt.Errorf("failed to start: %w", err)
		}

		// TODO:
		// c.log.Warn()

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(5 * time.Second):
		}
	}

}

func toInfo(i *api.Instance) InstanceInfo {
	return InstanceInfo{
		Name:         i.Name,
		Project:      i.Project,
		Location:     i.Location,
		Architecture: i.Architecture,
	}
}

func withReconnect[T any](c *Client, ctx context.Context, op func() (T, error)) (T, error) {
	return retry(ctx, func() (T, error) {
		result, err := op()
		if err != nil && isUnreachable(err) {
			conn, connErr := c.connect(c.rootCtx)
			if connErr != nil {
				return result, fmt.Errorf("reconnecting: %w", connErr)
			}

			// TODO: atomic swap
			c.server = conn
			return op()
		}
		return result, err
	}, isUnreachable,
	)
}

func retry[T any](ctx context.Context,
	op func() (result T, err error),
	shouldRetry func(error) bool,
) (T, error) {
	var result T
	var err error
	for range 3 {
		select {
		case <-ctx.Done():
			return result, ctx.Err()
		default:
			result, err = op()
			if err != nil && shouldRetry(err) {
				select {
				case <-ctx.Done():
					return result, ctx.Err()
				case <-time.After(1 * time.Second):
				}
			} else {
				return result, err
			}
		}
	}

	return result, err
}
