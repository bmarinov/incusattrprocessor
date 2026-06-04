package integrationtests

import (
	"os"
	"testing"
	"time"

	"github.com/bmarinov/otelcol-processor-incus/internal/incus"
)

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
		if _, ok := byKey[key]; !ok {
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

	if result.Name != info.Name {
		t.Errorf("Name: expected %q, got %q", info.Name, result.Name)
	}
	if result.Project != info.Project {
		t.Errorf("Project: expected %q, got %q", info.Project, result.Project)
	}
	if result.Architecture == "" {
		t.Error("Architecture: expected non-empty")
	}
}

func TestClient_Subscribe(t *testing.T) {
	c, env := setup(t)

	eventsCh := make(chan incus.InstanceEvent, 10)
	c.Subscribe(t.Context(), func(e incus.InstanceEvent) {
		if e.Action == incus.EventInstanceStarted {
			eventsCh <- e
		}
	}, func() {})

	inst := env.testInstance("foo-baz")
	waitEvent(t, eventsCh, inst.Name, 3*time.Second)
}

func TestClient_Subscribe_Stop(t *testing.T) {
	c, env := setup(t)

	eventsCh := make(chan incus.InstanceEvent, 10)
	c.Subscribe(t.Context(), func(e incus.InstanceEvent) {
		if e.Action == incus.EventInstanceStopped {
			eventsCh <- e
		}
	}, func() {})

	inst := env.testInstance("foo-stop")
	env.stopInstance(inst.Name)
	waitEvent(t, eventsCh, inst.Name, 3*time.Second)
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

	baseline := env.testInstance("reconnect-before")
	waitEvent(t, eventsCh, baseline.Name, 5*time.Second)

	socket := os.Getenv("INCUS_SOCKET")
	runOnVM(t, "sudo systemctl stop incus")
	runOnVM(t, "sudo systemctl start incus")
	waitIncusAPIReady(t, socket, 10*time.Second)

	select {
	case <-reconnected:
	case <-time.After(10 * time.Second):
		t.Fatal("onConnect not called after 10 sec")
	}

	after := env.testInstance("reconnect-after")
	waitEvent(t, eventsCh, after.Name, 10*time.Second)
}

func TestSubscribe_EventsRecoverAfterRestart(t *testing.T) {
	c, env := setup(t)
	socket := os.Getenv("INCUS_SOCKET")

	eventsCh := make(chan incus.InstanceEvent, 20)
	reconnected := make(chan struct{}, 5)
	c.Subscribe(t.Context(), func(e incus.InstanceEvent) {
		if e.Action == incus.EventInstanceStarted {
			eventsCh <- e
		}
	}, func() {
		select {
		case reconnected <- struct{}{}:
		default:
		}
	})

	select {
	case <-reconnected:
	case <-time.After(5 * time.Second):
		t.Fatal("initial websocket connection timeout")
	}

	env.testInstance("cycle-pre")
	waitEvent(t, eventsCh, "cycle-pre", 5*time.Second)

	runOnVM(t, "sudo systemctl stop incus")
	waitSocketDown(t, socket, 30*time.Second)

	runOnVM(t, "sudo systemctl start incus")
	waitIncusAPIReady(t, socket, 15*time.Second)

	select {
	case <-reconnected:
	case <-time.After(15 * time.Second):
		t.Fatal("websocket did not reconnect after restart")
	}

	for _, name := range []string{"cycle-post-1", "cycle-post-2", "cycle-post-3"} {
		env.testInstance(name)
		waitEvent(t, eventsCh, name, 10*time.Second)
	}
}
