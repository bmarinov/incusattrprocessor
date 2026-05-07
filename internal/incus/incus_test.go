package incus

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"

	incusclient "github.com/lxc/incus/v6/client"
	"go.uber.org/zap"
)

func TestReconnect_ConcurrentCalls(t *testing.T) {
	t.Parallel()
	const goroutines = 10

	var connectCalls atomic.Int32

	staleConn := &conn{srv: &fakeServer{}}

	c := &Client{
		log:             zap.NewNop(),
		reconnectPolicy: retryPolicy{attempts: 1},
		connect: func(_ context.Context) (incusclient.InstanceServer, error) {
			connectCalls.Add(1)
			return &fakeServer{}, nil
		},
	}
	c.rootCtx = context.Background()
	c.srv.Store(staleConn)

	start := make(chan struct{})
	var wg sync.WaitGroup
	for range goroutines {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			if err := c.reconnect(staleConn); err != nil {
				t.Errorf("reconnect: %v", err)
			}
		}()
	}

	close(start)
	wg.Wait()

	if n := connectCalls.Load(); n != 1 {
		t.Errorf("expected exactly 1 connect call got %d", n)
	}
}

// fakeServer embeds InstanceServer for tests
type fakeServer struct {
	incusclient.InstanceServer
}

func TestSplitLabel(t *testing.T) {
	tests := []struct {
		label       string
		wantProject string
		wantName    string
	}{
		{
			label:       "traefik-reverse-proxy",
			wantProject: "default",
			wantName:    "traefik-reverse-proxy",
		},
		{
			label:       "fooproject_sharing-martin",
			wantProject: "fooproject",
			wantName:    "sharing-martin",
		},
	}

	for _, tc := range tests {
		t.Run(tc.label, func(t *testing.T) {
			project, name := SplitLabel(tc.label)
			if project != tc.wantProject {
				t.Errorf("project: expected %q, got %q", tc.wantProject, project)
			}
			if name != tc.wantName {
				t.Errorf("name: expected %q, got %q", tc.wantName, name)
			}
		})
	}
}
