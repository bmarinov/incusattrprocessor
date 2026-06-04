package incus

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	incusclient "github.com/lxc/incus/v6/client"
	"go.uber.org/zap"
)

// unix socket no srv swap
func TestSubscribe_WebsocketRecoversWithoutConnSwap(t *testing.T) {
	socket := os.Getenv("INCUS_SOCKET")
	if socket == "" {
		t.Skip("INCUS_SOCKET not set")
	}
	if _, err := os.Stat(socket); err != nil {
		t.Skipf("INCUS_SOCKET not reachable: %v", err)
	}

	logger, err := zap.NewDevelopment()
	if err != nil {
		t.Fatal(err)
	}

	c := New(socket, logger, 3, time.Second)
	if err := c.Start(t.Context()); err != nil {
		t.Fatalf("client.Start: %v", err)
	}

	reconnected := make(chan struct{}, 1)
	c.Subscribe(t.Context(), func(InstanceEvent) {}, func() {
		select {
		case reconnected <- struct{}{}:
		default:
		}
	})

	// drain initial onConnect
	select {
	case <-reconnected:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for initial connection")
	}

	ptrBefore := c.srv.Load()

	vmSSH(t, "sudo bash -c 'systemctl stop incus.socket 2>/dev/null; systemctl stop incus'")
	vmSSH(t, "sudo systemctl start incus")
	waitSocketReady(t, socket, 15*time.Second)

	select {
	case <-reconnected:
	case <-time.After(15 * time.Second):
		t.Fatal("timed out waiting for websocket reconnect after daemon restart")
	}

	ptrAfter := c.srv.Load()

	if ptrBefore != ptrAfter {
		t.Error("srv pointer changed on websocket reconnect — atomic swap is not needed for event recovery")
	}
}

// Covers the long-running server scenario:
// daemon restarts -> in-flight API call ECONNREFUSED -> reconnect fires x n.
//
// TODO If this passes with the conn pointer unchanged then the conn swap can go.
func TestGetAllInstances_RecoverAfterDaemonRestart(t *testing.T) {
	socket := os.Getenv("INCUS_SOCKET")
	if socket == "" {
		t.Skip("INCUS_SOCKET not set")
	}
	if _, err := os.Stat(socket); err != nil {
		t.Skipf("INCUS_SOCKET not reachable: %v", err)
	}

	logger, err := zap.NewDevelopment()
	if err != nil {
		t.Fatal(err)
	}

	c := New(socket, logger, 10, time.Second)
	if err := c.Start(t.Context()); err != nil {
		t.Fatalf("client.Start: %v", err)
	}

	if _, err := c.GetAllInstances(t.Context()); err != nil {
		t.Fatalf("baseline: %v", err)
	}

	connBefore := c.srv.Load()

	// restart service even if the test fails.
	// the whole suite is brittle :/
	key := filepath.Join(os.Getenv("HOME"), ".cache/incus-test-vm/id_ed25519")
	sshArgs := []string{
		"-o", "StrictHostKeyChecking=no",
		"-o", "UserKnownHostsFile=/dev/null",
		"-o", "BatchMode=yes",
		"-i", key, "-p", "2299",
		"ubuntu@127.0.0.1",
	}
	t.Cleanup(func() {
		_ = exec.Command("ssh", append(sshArgs,
			"sudo bash -c 'systemctl start incus.socket 2>/dev/null; systemctl start incus'",
		)...).Run()
		waitSocketReady(t, socket, 20*time.Second)
	})

	vmSSH(t, "sudo bash -c 'systemctl stop incus.socket 2>/dev/null; systemctl stop incus'")
	waitSocketDown(t, socket, 30*time.Second)

	// Bring it back after a pause long enough for the reconnect attempt to fire
	// and fail (daemon still down), so the outer retry handles recovery instead.
	go func() {
		time.Sleep(time.Second)
		_ = exec.Command("ssh", append(sshArgs,
			"sudo bash -c 'systemctl start incus.socket 2>/dev/null; systemctl start incus'",
		)...).Run()
	}()

	if _, err := c.GetAllInstances(t.Context()); err != nil {
		t.Fatalf("GetAllInstances did not recover after daemon restart: %v", err)
	}

	if connAfter := c.srv.Load(); connAfter != connBefore {
		t.Error("conn pointer changed — atomic swap fired even though it was not needed for recovery")
	}
}

func vmSSH(t *testing.T, cmd string) {
	t.Helper()
	key := filepath.Join(os.Getenv("HOME"), ".cache/incus-test-vm/id_ed25519")
	if _, err := os.Stat(key); err != nil {
		t.Skipf("VM SSH key not found, skipping: %v", err)
	}
	out, err := exec.Command("ssh",
		"-o", "StrictHostKeyChecking=no",
		"-o", "UserKnownHostsFile=/dev/null",
		"-o", "BatchMode=yes",
		"-i", key,
		"-p", "2299",
		"ubuntu@127.0.0.1",
		cmd,
	).CombinedOutput()
	if err != nil {
		t.Fatalf("vmSSH %q: %v\n%s", cmd, err, out)
	}
}

func waitSocketDown(t *testing.T, socket string, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		_, err := incusclient.ConnectIncusUnixWithContext(ctx, socket, nil)
		cancel()
		if err != nil {
			return
		}
		time.Sleep(200 * time.Millisecond)
	}
	t.Fatalf("Incus daemon still responsive after %s — did not stop", timeout)
}

func waitSocketReady(t *testing.T, socket string, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if _, err := incusclient.ConnectIncusUnixWithContext(context.Background(), socket, nil); err == nil {
			return
		}
		time.Sleep(200 * time.Millisecond)
	}
	t.Fatalf("Incus socket not ready after %s", timeout)
}
