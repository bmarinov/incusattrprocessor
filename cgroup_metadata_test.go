package incusattrprocessor

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/bmarinov/otelcol-processor-incus/internal/incus"
)

func writeCgroup(t *testing.T, procRoot, pid, content string) {
	t.Helper()
	dir := filepath.Join(procRoot, pid)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "cgroup"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestCgroupMetadataSource_GetInstanceMetadata(t *testing.T) {
	t.Run("resolves container in default project", func(t *testing.T) {
		procRoot := t.TempDir()
		writeCgroup(t, procRoot, "100", "0::/lxc.payload.web-fe\n")
		src := &cgroupMetadataSource{procRoot: procRoot, client: &fakeInstanceLookup{info: incus.InstanceInfo{Location: "node-1"}}}

		got, err := src.GetInstanceMetadata(t.Context(), "100")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.Name != "web-fe" || got.Project != "default" || got.Location != "node-1" {
			t.Errorf("got %v, want {web-fe default node-1}", got)
		}
	})

	t.Run("resolves named-project container", func(t *testing.T) {
		procRoot := t.TempDir()
		writeCgroup(t, procRoot, "101", "0::/lxc.payload.fooproject_web\n")
		src := &cgroupMetadataSource{procRoot: procRoot, client: &fakeInstanceLookup{info: incus.InstanceInfo{Location: "node-2"}}}

		got, err := src.GetInstanceMetadata(t.Context(), "101")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.Name != "web" || got.Project != "fooproject" || got.Location != "node-2" {
			t.Errorf("got %v, want {web fooproject node-2}", got)
		}
	})

	t.Run("returns error when pid cgroup is not an LXC path", func(t *testing.T) {
		procRoot := t.TempDir()
		writeCgroup(t, procRoot, "300", "0::/system.slice/sshd.service\n")
		src := &cgroupMetadataSource{procRoot: procRoot, client: &fakeInstanceLookup{}}

		_, err := src.GetInstanceMetadata(t.Context(), "300")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("returns error when pid does not exist", func(t *testing.T) {
		src := &cgroupMetadataSource{procRoot: t.TempDir(), client: &fakeInstanceLookup{}}

		_, err := src.GetInstanceMetadata(t.Context(), "999")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("returns error when Incus API call fails", func(t *testing.T) {
		procRoot := t.TempDir()
		writeCgroup(t, procRoot, "400", "0::/lxc.payload.web-frontend\n")
		src := &cgroupMetadataSource{procRoot: procRoot, client: &fakeInstanceLookup{err: fmt.Errorf("connection refused")}}

		_, err := src.GetInstanceMetadata(t.Context(), "400")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

type fakeInstanceLookup struct {
	info incus.InstanceInfo
	err  error
}

func (f *fakeInstanceLookup) GetInstance(_ context.Context, _, _ string) (incus.InstanceInfo, error) {
	return f.info, f.err
}
