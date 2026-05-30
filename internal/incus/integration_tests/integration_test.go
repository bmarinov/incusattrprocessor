package integrationtests

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/bmarinov/otelcol-processor-incus/internal/incus"
	incusclient "github.com/lxc/incus/v7/client"
	"github.com/lxc/incus/v7/shared/api"
	"go.uber.org/zap"
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
	})

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
	})

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
	})

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
	c := incus.New(socket, zap.NewNop(), 3, time.Second)
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
