package integrationtests

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/bmarinov/otelcol-processor-incus/internal/incus"
	incusclient "github.com/lxc/incus/v6/client"
	"github.com/lxc/incus/v6/shared/api"
	"go.uber.org/zap"
)

var vmSSHKey = filepath.Join(os.Getenv("HOME"), ".cache/incus-test-vm/id_ed25519")

func TestMain(m *testing.M) {
	if os.Getenv("INCUS_SOCKET") == "" {
		os.Exit(0)
	}
	os.Exit(m.Run())
}

func setup(t *testing.T) (*incus.Client, *testEnv) {
	t.Helper()
	socket := os.Getenv("INCUS_SOCKET")
	if _, err := os.Stat(socket); err != nil {
		t.Skipf("INCUS_SOCKET %s not reachable (VM not running?): %v", socket, err)
	}

	srv, err := incusclient.ConnectIncusUnixWithContext(context.Background(), socket, nil)
	if err != nil {
		t.Fatalf("setup: connect: %v", err)
	}
	logger, err := zap.NewDevelopment()
	if err != nil {
		t.Fatal(err)
	}
	c := incus.New(socket, logger, 3, time.Second)
	if err := c.Start(t.Context()); err != nil {
		t.Fatalf("client.Start(): %v", err)
	}

	return c, &testEnv{t: t, srv: srv}
}

func setupHTTPS(t *testing.T) (*incus.Client, *testEnv) {
	t.Helper()
	cfg := incus.HTTPSConfig{
		URL:        requireEnv(t, "INCUS_HTTPS_URL"),
		ClientCert: readEnvFile(t, "INCUS_CLIENT_CERT"),
		ClientKey:  readEnvFile(t, "INCUS_CLIENT_KEY"),
		ServerCert: readEnvFile(t, "INCUS_SERVER_CERT"),
	}
	logger, err := zap.NewDevelopment()
	if err != nil {
		t.Fatal(err)
	}
	c := incus.NewHTTPS(cfg, logger, 3, time.Second)
	if err := c.Start(t.Context()); err != nil {
		t.Fatalf("client.Start() over HTTPS: %v", err)
	}
	_, env := setup(t)
	return c, env
}

func requireEnv(t *testing.T, name string) string {
	t.Helper()
	v := os.Getenv(name)
	if v == "" {
		t.Skipf("%s not set", name)
	}
	return v
}

func readEnvFile(t *testing.T, name string) string {
	t.Helper()
	path := requireEnv(t, name)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %s=%s: %v", name, path, err)
	}
	return string(data)
}

type testEnv struct {
	t   *testing.T
	srv incusclient.InstanceServer
}

func (e *testEnv) testInstance(name string) incus.InstanceInfo {
	e.t.Helper()
	info := e.createInstance(name)
	e.startInstance(info.Name)
	return info
}

func forceDelete(srv incusclient.InstanceServer, name string) {
	proj := srv.UseProject("default")
	if op, err := proj.UpdateInstanceState(name, api.InstanceStatePut{Action: "stop", Force: true}, ""); err == nil {
		_ = op.WaitContext(context.Background())
	}
	if op, err := proj.DeleteInstance(name); err == nil {
		_ = op.WaitContext(context.Background())
	}
}

func (e *testEnv) createInstance(name string) incus.InstanceInfo {
	e.t.Helper()

	forceDelete(e.srv, name)

	op, err := e.srv.UseProject("default").CreateInstance(api.InstancesPost{
		Name: name,
		Type: api.InstanceTypeContainer,
		Source: api.InstanceSource{
			Type:     "image",
			Server:   "https://images.linuxcontainers.org",
			Protocol: "simplestreams",
			Alias:    "alpine/edge",
		},
	})
	if err != nil {
		e.t.Fatalf("createInstance %s: %v", name, err)
	}
	if err := op.WaitContext(e.t.Context()); err != nil {
		e.t.Fatalf("createInstance %s: wait: %v", name, err)
	}

	e.t.Cleanup(func() { forceDelete(e.srv, name) })

	return incus.InstanceInfo{Name: name, Project: "default"}
}

func (e *testEnv) startInstance(name string) {
	e.t.Helper()
	op, err := e.srv.UseProject("default").UpdateInstanceState(
		name, api.InstanceStatePut{Action: "start"}, "",
	)
	if err != nil {
		e.t.Fatalf("startInstance %s: %v", name, err)
	}
	if err := op.WaitContext(e.t.Context()); err != nil {
		e.t.Fatalf("startInstance %s: wait: %v", name, err)
	}
}

func (e *testEnv) stopInstance(name string) {
	e.t.Helper()
	op, err := e.srv.UseProject("default").UpdateInstanceState(
		name, api.InstanceStatePut{Action: "stop", Force: true}, "",
	)
	if err != nil {
		e.t.Fatalf("stopInstance %s: %v", name, err)
	}
	if err := op.WaitContext(e.t.Context()); err != nil {
		e.t.Fatalf("stopInstance %s: wait: %v", name, err)
	}
}

func (e *testEnv) renameInstance(oldName, newName string) {
	e.t.Helper()
	op, err := e.srv.UseProject("default").RenameInstance(
		oldName, api.InstancePost{Name: newName},
	)
	if err != nil {
		e.t.Fatalf("renameInstance %s to %s: %v", oldName, newName, err)
	}
	if err := op.WaitContext(e.t.Context()); err != nil {
		e.t.Fatalf("renameInstance %s to %s: wait: %v", oldName, newName, err)
	}
	e.t.Cleanup(func() { forceDelete(e.srv, newName) })
}

func waitEvent(t *testing.T, ch <-chan incus.InstanceEvent, name string, timeout time.Duration) {
	t.Helper()
	deadline := time.After(timeout)
	for {
		select {
		case e := <-ch:
			if e.Name == name {
				return
			}
		case <-deadline:
			t.Fatalf("timed out waiting for event for instance %q", name)
		}
	}
}

func waitIncusAPIReady(t *testing.T, socket string, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if _, err := incusclient.ConnectIncusUnixWithContext(context.Background(), socket, nil); err == nil {
			return
		}
		time.Sleep(200 * time.Millisecond)
	}
	t.Fatalf("Incus API not ready after %s", timeout)
}

func waitSocketDown(t *testing.T, socket string, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		// Short per-attempt context so a hanging daemon doesn't consume the whole budget.
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

func runOnVM(t *testing.T, cmd string) {
	t.Helper()
	if _, err := os.Stat(vmSSHKey); err != nil {
		t.Fatalf("VM SSH key not found: %v", err)
	}
	out, err := exec.Command("ssh",
		"-o", "StrictHostKeyChecking=no",
		"-o", "UserKnownHostsFile=/dev/null",
		"-o", "BatchMode=yes",
		"-i", vmSSHKey,
		"-p", "2299",
		"ubuntu@127.0.0.1",
		cmd,
	).CombinedOutput()
	if err != nil {
		t.Fatalf("exec in vm: %q: %v\n%s", cmd, err, out)
	}
}
