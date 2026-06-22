package integrationtests

import (
	"testing"
	"time"

	"github.com/bmarinov/incusattrprocessor/internal/incus"
)

func TestHTTPS_GetAllInstances(t *testing.T) {
	c, env := setupHTTPS(t)

	a := env.testInstance("https-a")
	b := env.testInstance("https-b")

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

func TestHTTPS_GetInstance(t *testing.T) {
	c, env := setupHTTPS(t)

	info := env.testInstance("https-get")

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

func TestHTTPS_Subscribe(t *testing.T) {
	c, env := setupHTTPS(t)

	eventsCh := make(chan incus.InstanceEvent, 10)
	c.Subscribe(t.Context(), func(e incus.InstanceEvent) {
		if e.Action == incus.EventInstanceStarted {
			eventsCh <- e
		}
	}, func() {})

	inst := env.testInstance("https-sub")
	waitEvent(t, eventsCh, inst.Name, 5*time.Second)
}
