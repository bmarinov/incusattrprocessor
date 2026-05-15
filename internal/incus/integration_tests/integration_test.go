package integrationtests

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/bmarinov/otelcol-processor-incus/internal/incus"
	incusclient "github.com/lxc/incus/v6/client"
	"github.com/lxc/incus/v6/shared/api"
	"go.uber.org/zap"
)

func TestMain(m *testing.M) {
	if os.Getenv("INCUS_SOCKET") == "" {
		os.Exit(0)
	}
	os.Exit(m.Run())
}

func TestClient_GetAllInstances(t *testing.T) {
	c := setup(t)

	a := testInstance(t, "test-a")
	b := testInstance(t, "test-b")

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
	c := setup(t)

	info := testInstance(t, "foozy-bar")

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

func setup(t *testing.T) *incus.Client {
	t.Helper()
	socket := os.Getenv("INCUS_SOCKET")
	if _, err := os.Stat(socket); err != nil {
		t.Skipf("INCUS_SOCKET %s not reachable (VM not running?): %v", socket, err)
	}
	c := incus.New(socket, zap.NewNop(), 3, time.Second)
	err := c.Start(t.Context())
	if err != nil {
		t.Fatalf("client.Start(): %v", err)
	}
	return c
}

// testInstance creates a container in the default project.
// Registers a cleanup to delete it after the test.
func testInstance(t *testing.T, name string) incus.InstanceInfo {
	t.Helper()
	socket := os.Getenv("INCUS_SOCKET")

	const (
		project = "default"
	)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	t.Cleanup(cancel)

	srv, err := incusclient.ConnectIncusUnixWithContext(ctx, socket, nil)
	if err != nil {
		t.Fatalf("createTestInstance: connect: %v", err)
	}
	proj := srv.UseProject(project)

	// Delete leftover from a previous run:
	if op, err := proj.DeleteInstance(name); err == nil {
		_ = op.WaitContext(ctx)
	}

	op, err := proj.CreateInstance(api.InstancesPost{
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
		t.Fatalf("createTestInstance: create: %v", err)
	}
	if err := op.WaitContext(ctx); err != nil {
		t.Fatalf("createTestInstance: wait: %v", err)
	}

	t.Cleanup(func() {
		if op, err := proj.DeleteInstance(name); err == nil {
			_ = op.WaitContext(ctx)
		}
	})

	return incus.InstanceInfo{Name: name, Project: project}
}
