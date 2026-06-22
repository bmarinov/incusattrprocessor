package integrationtests

import (
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/bmarinov/incusattrprocessor/internal/incus"
	"go.uber.org/zap"
)

// Tests try to cover the long-running server scenario:
// daemon restarts, in-flight API call gets ECONNREFUSED,
// and the retry loop recovers without any conn instance swap.

func TestSubscribe_WebsocketRecoversAfterDaemonRestart(t *testing.T) {
	c, _ := setup(t)

	reconnected := make(chan struct{}, 1)
	c.Subscribe(t.Context(), func(incus.InstanceEvent) {}, func() {
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

	runOnVM(t, "sudo systemctl stop incus")
	runOnVM(t, "sudo systemctl start incus")
	waitIncusAPIReady(t, os.Getenv("INCUS_SOCKET"), 15*time.Second)

	select {
	case <-reconnected:
	case <-time.After(15 * time.Second):
		t.Fatal("timed out waiting for websocket reconnect after daemon restart")
	}
}

func TestGetAllInstances_RecoverAfterDaemonRestart(t *testing.T) {
	socket := os.Getenv("INCUS_SOCKET")
	if _, err := os.Stat(socket); err != nil {
		t.Skipf("INCUS_SOCKET %s not reachable (VM not running?): %v", socket, err)
	}

	logger, err := zap.NewDevelopment()
	if err != nil {
		t.Fatal(err)
	}
	c := incus.New(socket, logger, 10, time.Second)
	if err := c.Start(t.Context()); err != nil {
		t.Fatalf("client.Start: %v", err)
	}

	if _, err := c.GetAllInstances(t.Context()); err != nil {
		t.Fatalf("baseline: %v", err)
	}

	sshArgs := []string{
		"-o", "StrictHostKeyChecking=no",
		"-o", "UserKnownHostsFile=/dev/null",
		"-o", "BatchMode=yes",
		"-i", vmSSHKey, "-p", "2299",
		"ubuntu@127.0.0.1",
	}
	t.Cleanup(func() {
		_ = exec.Command("ssh", append(sshArgs, "sudo systemctl start incus")...).Run()
		waitIncusAPIReady(t, socket, 20*time.Second)
	})

	runOnVM(t, "sudo systemctl stop incus")
	waitSocketDown(t, socket, 30*time.Second)

	// Bring it back after a pause long enough for a retry to fire and fail while
	// the daemon is still down, so recovery exercises the full retry loop.
	go func() {
		time.Sleep(time.Second)
		_ = exec.Command("ssh", append(sshArgs, "sudo systemctl start incus")...).Run()
	}()

	if _, err := c.GetAllInstances(t.Context()); err != nil {
		t.Fatalf("GetAllInstances did not recover after daemon restart: %v", err)
	}
}
