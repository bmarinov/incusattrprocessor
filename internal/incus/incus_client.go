package incus

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	incusclient "github.com/lxc/incus/v6/client"
	"github.com/lxc/incus/v6/shared/api"
	"go.uber.org/zap"
)

// Client looks up Incus instance metadata via the local Unix socket.
type Client struct {
	// pointer to the connection with the Incus daemon
	srv       atomic.Pointer[conn]
	connect   connectFunc
	connectMu sync.Mutex
	// done is closed on processor context cancel.
	done            <-chan struct{}
	log             *zap.Logger
	reconnectPolicy retryPolicy
}

// conn wraps the InstanceServer for the atomic swap.
type conn struct {
	srv incusclient.InstanceServer
}

type connectFunc func(ctx context.Context) (incusclient.InstanceServer, error)

type InstanceInfo struct {
	Name         string
	Project      string
	Location     string
	Architecture string
	// TODO: cpu limits
}

func New(socketPath string,
	logger *zap.Logger,
	retryAttempts int,
	retryDelay time.Duration,
) *Client {
	return &Client{
		connect: func(ctx context.Context) (incusclient.InstanceServer, error) {
			conn, err := incusclient.ConnectIncusUnixWithContext(ctx, socketPath, nil)
			if err != nil {
				return nil, fmt.Errorf("connecting to incus daemon: %w", err)
			}
			return conn, nil
		},
		log: logger,
		reconnectPolicy: retryPolicy{
			attempts: retryAttempts,
			delay:    retryDelay,
		},
	}
}

func (c *Client) Start(ctx context.Context) error {
	c.done = ctx.Done()

	for {
		srvConn, err := retry(ctx,
			c.reconnectPolicy,
			func() (result incusclient.InstanceServer, err error) {
				return c.connect(ctx)
			}, isUnreachable)

		if err == nil {
			c.srv.Store(&conn{srvConn})
			return nil
		}
		if !isUnreachable(err) {
			return fmt.Errorf("failed to start: %w", err)
		}

		c.log.Warn("Incus daemon unreachable, will retry", zap.Error(err))
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(5 * time.Second):
		}
	}
}

// GetInstance returns the cluster member (location) hosting the instance.
func (c *Client) GetInstance(ctx context.Context, project, name string) (InstanceInfo, error) {
	inst, err := withReconnect(c, ctx, func(srv incusclient.InstanceServer) (*api.Instance, error) {
		inst, _, err := srv.UseProject(project).GetInstance(name)
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
	instances, err := withReconnect(c, ctx, func(srv incusclient.InstanceServer) ([]api.Instance, error) {
		return srv.GetInstancesAllProjects(api.InstanceTypeAny)
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

// Subscribe implements [metadata.InstanceEvents].
func (c *Client) Subscribe(ctx context.Context, callback func(e InstanceEvent)) {
	panic("unimplemented")
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

func toInfo(i *api.Instance) InstanceInfo {
	return InstanceInfo{
		Name:         i.Name,
		Project:      i.Project,
		Location:     i.Location,
		Architecture: i.Architecture,
	}
}

func withReconnect[T any](c *Client,
	ctx context.Context,
	op func(incusclient.InstanceServer) (T, error)) (T, error) {
	return retry(ctx, c.reconnectPolicy, func() (T, error) {

		currentConn := c.srv.Load()
		result, err := op(currentConn.srv)
		if err != nil && isUnreachable(err) {
			err = c.reconnect(currentConn)
			if err != nil {
				return result, fmt.Errorf("reconnecting: %w", err)
			}

			return op(c.srv.Load().srv)
		}
		return result, err
	},
		isUnreachable,
	)
}

// reconnect attempts to connect and swaps the underlying server connection.
func (c *Client) reconnect(current *conn) error {
	c.connectMu.Lock()
	defer c.connectMu.Unlock()

	if c.srv.Load() != current {
		// already reconnected
		return nil
	}

	connectCtx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		select {
		case <-c.done:
			cancel()
		case <-connectCtx.Done():
		}
	}()

	srv, err := c.connect(connectCtx)
	if err != nil {
		return err
	}

	c.srv.Store(&conn{srv: srv})
	return nil
}
