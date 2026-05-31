package metadata

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/bmarinov/otelcol-processor-incus/internal/incus"
	"go.uber.org/zap"
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
		src := &CgroupMetadataSource{procRoot: procRoot, lookup: &fakeInstanceLookup{info: incus.InstanceInfo{Location: "node-1"}}}

		got, err := src.GetInstanceMetadata(t.Context(), "100")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.Location != "node-1" {
			t.Errorf("Location: want %q, got %q", "node-1", got.Location)
		}
	})

	t.Run("resolves container in non-default project", func(t *testing.T) {
		procRoot := t.TempDir()
		writeCgroup(t, procRoot, "101", "0::/lxc.payload.fooproject_web\n")
		src := &CgroupMetadataSource{procRoot: procRoot, lookup: &fakeInstanceLookup{info: incus.InstanceInfo{Location: "node-2"}}}

		got, err := src.GetInstanceMetadata(t.Context(), "101")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.Location != "node-2" {
			t.Errorf("Location: want %q, got %q", "node-2", got.Location)
		}
	})

	t.Run("returns error when pid cgroup is not an LXC path", func(t *testing.T) {
		procRoot := t.TempDir()
		writeCgroup(t, procRoot, "300", "0::/system.slice/sshd.service\n")
		src := &CgroupMetadataSource{procRoot: procRoot, lookup: &fakeInstanceLookup{}}

		_, err := src.GetInstanceMetadata(t.Context(), "300")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("returns error when pid does not exist", func(t *testing.T) {
		src := &CgroupMetadataSource{procRoot: t.TempDir(), lookup: &fakeInstanceLookup{}}

		_, err := src.GetInstanceMetadata(t.Context(), "999")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("returns error when Incus API call fails", func(t *testing.T) {
		procRoot := t.TempDir()
		writeCgroup(t, procRoot, "400", "0::/lxc.payload.web-frontend\n")
		src := &CgroupMetadataSource{procRoot: procRoot, lookup: &fakeInstanceLookup{err: errors.New("connection refused")}}

		_, err := src.GetInstanceMetadata(t.Context(), "400")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestCache_GetInstance(t *testing.T) {
	t.Run("cache hit", func(t *testing.T) {
		project, instance := "default", "runner"

		// prepare
		seed := incus.InstanceInfo{
			Name:         instance,
			Project:      project,
			Location:     "node-0",
			Architecture: "amd64",
		}
		c, _, _ := setupCache(seed)
		_ = c.Start(t.Context())

		result, err := c.GetInstance(t.Context(), project, instance)
		if err != nil {
			t.Fatal(err)
		}
		if result.Architecture != seed.Architecture {
			t.Errorf("expected %q got %q", seed.Architecture, result.Architecture)
		}
		if result.Location != seed.Location {
			t.Errorf("expected %q got %q", seed.Location, result.Location)
		}
	})
	t.Run("cache miss", func(t *testing.T) {
		c, _, _ := setupCache()
		unknownInstance := "unknown_new"
		project := "blap"

		result, err := c.GetInstance(t.Context(),
			project, unknownInstance,
		)

		if err != nil {
			t.Fatalf("expected no error for cache miss, got %v", err)
		}
		if result.Name != unknownInstance || result.Project != project {
			t.Errorf("expected partial result on miss, got %+v", result)
		}
	})
	t.Run("concurrent cache misses", func(t *testing.T) {
		c, _, _ := setupCache()

		if err := c.Start(t.Context()); err != nil {
			t.Fatal(err)
		}

		var wg sync.WaitGroup
		for range 10 {
			wg.Add(1)
			go func() {
				defer wg.Done()
				_, _ = c.GetInstance(t.Context(), "default", "new-container")
			}()
		}
		wg.Wait()
	})
	t.Run("purge events clear the cache entry", func(t *testing.T) {
		for _, action := range []string{
			incus.EventInstanceStopped,
			incus.EventInstanceShutdown,
			incus.EventInstanceDeleted,
		} {
			t.Run(action, func(t *testing.T) {
				initial := incus.InstanceInfo{
					Name:         "once",
					Project:      "test",
					Architecture: "amd64",
					Location:     "none",
				}
				c, _, events := setupCache(initial)
				_ = c.Start(t.Context())

				events.Push(incus.InstanceEvent{Name: initial.Name, Project: initial.Project, Action: action})

				result, err := c.GetInstance(t.Context(), initial.Project, initial.Name)
				if err != nil {
					t.Fatal(err)
				}
				if result.Architecture != "" || result.Location != "" {
					t.Errorf("expected empty entry after purge, got %+v", result)
				}
			})
		}
	})
	t.Run("update events refresh the cache entry", func(t *testing.T) {
		for _, action := range []string{incus.EventInstanceStarted, incus.EventInstanceRestarted} {
			t.Run(action, func(t *testing.T) {
				initial := incus.InstanceInfo{Name: "tripple", Project: "projects", Architecture: "amd64", Location: "none"}
				c, lookup, events := setupCache(initial)
				_ = c.Start(t.Context())
				lookup.info.Location = "new-node"

				events.Push(incus.InstanceEvent{Name: initial.Name, Project: initial.Project, Action: action})
				lookup.WaitFetch(t)

				result, err := c.GetInstance(t.Context(), initial.Project, initial.Name)
				if err != nil {
					t.Fatal(err)
				}
				if result.Location != "new-node" {
					t.Errorf("expected new Location after update, got %+v", result)
				}
			})
		}
	})
	t.Run("rename event evicts old name and fetches new", func(t *testing.T) {
		initial := incus.InstanceInfo{
			Name:         "old-name",
			Project:      "projects",
			Architecture: "amd64",
			Location:     "none",
		}
		c, lookup, events := setupCache(initial)
		_ = c.Start(t.Context())

		// Architecture is the canary: it is empty for a cache miss.
		lookup.info.Architecture = "aarch64"

		const newName = "new-name"
		events.Push(incus.InstanceEvent{
			Name:    newName,
			OldName: initial.Name,
			Project: initial.Project,
			Action:  incus.EventInstanceRenamed,
		})
		lookup.WaitFetch(t)

		result, err := c.GetInstance(t.Context(), initial.Project, newName)
		if err != nil {
			t.Fatal(err)
		}
		if result.Architecture == "" {
			t.Errorf("expected fully fetched entry for new name, got partial result: %+v", result)
		}
		// old name should be evicted
		old, err := c.GetInstance(t.Context(), initial.Project, initial.Name)
		if err != nil {
			t.Fatal(err)
		}
		if old.Architecture != "" || old.Location != "" {
			t.Errorf("expected empty entry for old name after rename, got %+v", old)
		}
	})
}

func TestCache_RebuildAfterReconnect(t *testing.T) {
	warmupData := []incus.InstanceInfo{
		{Name: "stale", Project: "p", Architecture: "amd64"},
	}

	events := &fakeEventSource{}
	c := NewCache(
		&fakeInstanceLookup{fetched: make(chan struct{}, 1)},
		events,
		func(ctx context.Context) ([]incus.InstanceInfo, error) { return warmupData, nil },
		zap.NewNop(),
	)
	if err := c.Start(t.Context()); err != nil {
		t.Fatal(err)
	}

	got, err := c.GetInstance(t.Context(), "p", "stale")
	if err != nil {
		t.Fatal(err)
	}
	if got.Architecture != "amd64" {
		t.Fatalf("expected stale instance in cache after initial warmup, got %+v", got)
	}

	// Swap data for next warmup:
	warmupData = []incus.InstanceInfo{
		{Name: "fresh", Project: "p", Architecture: "aarch64"},
	}
	events.SignalConnected()

	stale, _ := c.GetInstance(t.Context(), "p", "stale")
	if stale.Architecture != "" {
		t.Errorf("stale entry not evicted after reconnect, got %+v", stale)
	}
	fresh, _ := c.GetInstance(t.Context(), "p", "fresh")
	if fresh.Architecture != "aarch64" {
		t.Errorf("fresh entry not in cache after reconnect, got %+v", fresh)
	}
}

func TestCache_Startup(t *testing.T) {
	seed := incus.InstanceInfo{
		Name:         "blap",
		Project:      "foo",
		Location:     "df",
		Architecture: "aarch64",
	}

	c, _, _ := setupCache(seed)

	t.Run("retrieve after warmup", func(t *testing.T) {
		err := c.Start(t.Context())
		if err != nil {
			t.Fatal(err)
		}

		result, err := c.GetInstance(t.Context(), seed.Project, seed.Name)
		if err != nil {
			t.Fatal(err)
		}

		if result.Architecture != seed.Architecture || result.Location != seed.Location {
			t.Errorf("expected %+v got %+v", seed, result)
		}
	})
}

func setupCache(seed ...incus.InstanceInfo) (*Cache, *fakeInstanceLookup, *fakeEventSource) {
	l := fakeInstanceLookup{fetched: make(chan struct{}, 1)}
	events := fakeEventSource{}
	c := NewCache(
		&l,
		&events,
		func(ctx context.Context) ([]incus.InstanceInfo, error) { return seed, nil },
		zap.NewNop(),
	)
	return c, &l, &events
}

type fakeInstanceLookup struct {
	info    incus.InstanceInfo
	err     error
	fetched chan struct{}
}

func (f *fakeInstanceLookup) GetInstance(_ context.Context, project, name string) (incus.InstanceInfo, error) {
	select {
	case f.fetched <- struct{}{}:
	default:
	}
	info := f.info
	info.Name = name
	info.Project = project
	return info, f.err
}

// WaitFetch blocks until fresh instance info is fetched with GetInstance.
func (f *fakeInstanceLookup) WaitFetch(t *testing.T) {
	t.Helper()
	select {
	case <-f.fetched:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for background instance fetch")
	}
}

type fakeEventSource struct {
	subscriptions []func(incus.InstanceEvent)
	onConnect     func()
}

// Subscribe implements [InstanceEvents].
func (s *fakeEventSource) Subscribe(_ context.Context, cb func(e incus.InstanceEvent), onConnect func()) {
	s.subscriptions = append(s.subscriptions, cb)
	s.onConnect = onConnect
	if onConnect != nil {
		onConnect()
	}
}

// SignalConnected simulates the incus.Client signalling on established connection.
func (s *fakeEventSource) SignalConnected() {
	if s.onConnect != nil {
		s.onConnect()
	}
}

func (s *fakeEventSource) Push(e incus.InstanceEvent) {
	if e.Action == incus.EventInstanceRenamed && e.OldName == "" {
		panic("test: rename event requires OldName to be set")
	}
	for _, cb := range s.subscriptions {
		cb(e)
	}
}
