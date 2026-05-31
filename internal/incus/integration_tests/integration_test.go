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

var (
	vmRunDir = filepath.Join(os.Getenv("HOME"), ".cache/incus-test-vm/run")
	vmSSHKey = filepath.Join(os.Getenv("HOME"), ".cache/incus-test-vm/id_ed25519")
)

func TestMain(m *testing.M) {
	if os.Getenv("INCUS_SOCKET") == "" {
		os.Exit(0)
	}
	os.Exit(m.Run())
}

func TestClient_GetAllInstances(t *testing.T) {
	c, env := setup(t)

	a := env.testInstance("test-a")
	b := env.testInstance("test-b")

	instances, err := c.GetAllInstances(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	byKey := make(map[string]incus.InstanceInfo)
	for _, inst := range instances {
		byKey[inst.Project+"/"+inst.Name] = inst
	}

	for _, expected := range []incus.InstanceInfo{a, b} {
		key := expected.Project + "/" + expected.Name
		_, ok := byKey[key]
		if !ok {
			t.Errorf("instance %s missing from result", key)
		}
	}
}

func TestClient_GetInstance(t *testing.T) {
	c, env := setup(t)

	info := env.testInstance("foozy-bar")

	result, err := c.GetInstance(t.Context(), info.Project, info.Name)
	if err != nil {
		t.Fatal(err)
	}

	if result.Name != info.Name ||
		result.Project != info.Project ||
		result.Architecture == "" {
		t.Errorf("unexpected field vals in result: %q", result)
	}
}

func TestClient_Subscribe(t *testing.T) {
	c, env := setup(t)

	eventsCh := make(chan incus.InstanceEvent, 10)
	c.Subscribe(t.Context(), func(e incus.InstanceEvent) {
		eventsCh <- e
	}, func() {},
	)

	newInst := env.testInstance("foo-baz")

	var found *incus.InstanceEvent
	timeout := time.After(3 * time.Second)

	for found == nil {
		select {
		case msg := <-eventsCh:
			if msg.Action == incus.EventInstanceStarted {
				found = &msg
			}
		case <-timeout:
			t.Fatal("timed out waiting for event")
		}
	}

	if found.Name != newInst.Name || found.Project != newInst.Project {
		t.Errorf("unexpected values for container event: %+v", found)
	}
}

func TestClient_Subscribe_Stop(t *testing.T) {
	c, env := setup(t)

	eventsCh := make(chan incus.InstanceEvent, 10)
	c.Subscribe(t.Context(), func(e incus.InstanceEvent) {
		eventsCh <- e
	}, func() {})

	inst := env.testInstance("foo-stop")
	env.stopInstance(inst.Name)

	var found *incus.InstanceEvent
	timeout := time.After(3 * time.Second)

	for found == nil {
		select {
		case msg := <-eventsCh:
			if msg.Action == incus.EventInstanceStopped {
				found = &msg
			}
		case <-timeout:
			t.Fatal("timed out waiting for event")
		}
	}

	if found.Name != inst.Name || found.Project != inst.Project {
		t.Errorf("unexpected values for container event: %+v", found)
	}
}

func TestClient_Subscribe_Rename(t *testing.T) {
	c, env := setup(t)

	instance := env.createInstance("old-name")

	eventsCh := make(chan incus.InstanceEvent, 10)
	c.Subscribe(t.Context(), func(e incus.InstanceEvent) {
		if e.Action == incus.EventInstanceRenamed {
			eventsCh <- e
		}
	}, func() {})

	const newName = "new-foo"
	env.renameInstance(instance.Name, newName)

	select {
	case e := <-eventsCh:
		if e.Name != newName {
			t.Errorf("Name: expected %q (new name), got %q", newName, e.Name)
		}
		if e.OldName != instance.Name {
			t.Errorf("OldName: expected %q got %q", instance.Name, e.OldName)
		}
		if e.Project != "default" {
			t.Errorf("Project: expected %q got %q", "default", e.Project)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for event")
	}
}

func TestClient_Subscribe_ReceiveAfterReconnect(t *testing.T) {
	c, env := setup(t)
	eventsCh := make(chan incus.InstanceEvent, 20)
	reconnected := make(chan struct{}, 1)
	c.Subscribe(t.Context(),
		func(e incus.InstanceEvent) {
			if e.Action == incus.EventInstanceStarted {
				eventsCh <- e
			}
		},
		func() { reconnected <- struct{}{} },
	)

	// verify
	baseline := env.testInstance("reconnect-before")
	waitEvent(t, eventsCh, baseline.Name, 5*time.Second)

	socket := os.Getenv("INCUS_SOCKET")

	// act
	runOnVM(t, "sudo systemctl stop incus")
	runOnVM(t, "sudo systemctl start incus")
	waitIncusAPIReady(t, socket, 10*time.Second)

	select {
	case <-reconnected:
	case <-time.After(10 * time.Second):
		t.Fatal("onConnect not called after 10 sec")
	}

	// assert
	after := env.testInstance("reconnect-after")
	waitEvent(t, eventsCh, after.Name, 10*time.Second)
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

type testEnv struct {
	t   *testing.T
	srv incusclient.InstanceServer
}

// testInstance creates and starts a container with name.
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

// createInstance creates a stopped container.
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

// renameInstance renames a stopped instance.
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

// waitEvent drains eventsCh until an event with the given name arrives or timeout fires.
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

// waitIncusAPIReady polls until the Incus API responds (daemon up).
func waitIncusAPIReady(t *testing.T, socket string, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		_, err := incusclient.ConnectIncusUnixWithContext(context.Background(), socket, nil)
		if err == nil {
			return
		}
		time.Sleep(200 * time.Millisecond)
	}
	t.Fatalf("waiting for Incus API: not ready after %s", timeout)
}

// runOnVM runs a command on the Incus VM over SSH.
func runOnVM(t *testing.T, cmd string) {
	t.Helper()
	if _, err := os.Stat(vmSSHKey); err != nil {
		t.Fatalf("VM SSH key not found: cannot run command: %v", err)
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
