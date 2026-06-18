package incusattrprocessor

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/bmarinov/otelcol-processor-incus/internal/incus"
	"github.com/bmarinov/otelcol-processor-incus/internal/metadata"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pprofile"
	"go.opentelemetry.io/collector/processor"
	"go.uber.org/zap"
)

func writeCgroup(t testing.TB, procRoot, pid, content string) {
	t.Helper()
	dir := filepath.Join(procRoot, pid)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "cgroup"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func warmupWith(seed ...incus.InstanceInfo) metadata.WarmupFunc {
	return func(ctx context.Context) ([]incus.InstanceInfo, error) {
		return seed, nil
	}
}

func TestIncusAttrProcessor_processProfiles(t *testing.T) {
	t.Run("adds container metadata to resource when pid matches a running instance", func(t *testing.T) {
		procRoot := t.TempDir()
		writeCgroup(t, procRoot, "1122", "0::/lxc.payload.container-foo\n")
		seed := incus.InstanceInfo{
			Name:     "container-foo",
			Project:  "default",
			Location: "node-0",
		}
		cacheLookup := metadata.NewCache(nil,
			&noopEventSource{},
			warmupWith(seed),
			zap.NewNop())
		_ = cacheLookup.Start(t.Context())
		src := metadata.NewSource(cacheLookup, procRoot)
		p := newIncusAttrProcessor(
			nopSettings(),
			&processorConfig{},
			src,
			noStart)

		// act
		pd, _ := newProfilesWithPID(1122)
		got, err := p.processProfiles(context.Background(), pd)
		if err != nil {
			t.Fatalf("processProfiles returned unexpected error: %v", err)
		}

		attrs := got.ResourceProfiles().At(0).Resource().Attributes()
		assertAttr(t, attrs, attrInstanceName, "container-foo")
		assertAttr(t, attrs, attrInstanceProject, "default")
		assertAttr(t, attrs, attrInstanceLocation, "node-0")
	})

	t.Run("leaves resource unchanged when pid is not in a container", func(t *testing.T) {
		procRoot := t.TempDir()
		writeCgroup(t, procRoot, "300", "0::/system.slice/sshd.service\n")

		cacheLookup := metadata.NewCache(nil,
			&noopEventSource{},
			warmupWith(),
			zap.NewNop())
		_ = cacheLookup.Start(t.Context())
		src := metadata.NewSource(cacheLookup, procRoot)
		p := newIncusAttrProcessor(nopSettings(), &processorConfig{}, src, noStart)

		pd, _ := newProfilesWithPID(300)
		got, err := p.processProfiles(context.Background(), pd)
		if err != nil {
			t.Fatalf("processProfiles returned unexpected error: %v", err)
		}
		attrs := got.ResourceProfiles().At(0).Resource().Attributes()
		if _, ok := attrs.Get(attrInstanceName); ok {
			t.Errorf("expected no %s attribute on non-container pid", attrInstanceName)
		}
	})

	t.Run("leaves resource unchanged when pid has no cgroup entry", func(t *testing.T) {
		src := metadata.NewSource(
			metadata.NewCache(nil,
				&noopEventSource{},
				warmupWith(),
				zap.NewNop()),
			t.TempDir())
		p := newIncusAttrProcessor(nopSettings(), &processorConfig{}, src, noStart)

		pd, _ := newProfilesWithPID(9999)
		got, err := p.processProfiles(context.Background(), pd)
		if err != nil {
			t.Fatalf("processProfiles returned unexpected error: %v", err)
		}
		attrs := got.ResourceProfiles().At(0).Resource().Attributes()
		if _, ok := attrs.Get(attrInstanceName); ok {
			t.Errorf("expected no %s attribute for unknown pid", attrInstanceName)
		}
	})

	t.Run("leaves resource with no pid attr unchanged", func(t *testing.T) {
		src := metadata.NewSource(metadata.NewCache(nil, &noopEventSource{}, warmupWith(), zap.NewNop()),
			t.TempDir())
		p := newIncusAttrProcessor(nopSettings(), &processorConfig{}, src, noStart)

		pd := pprofile.NewProfiles()
		pd.ResourceProfiles().AppendEmpty()
		got, err := p.processProfiles(context.Background(), pd)
		if err != nil {
			t.Fatalf("processProfiles returned unexpected error: %v", err)
		}
		attrs := got.ResourceProfiles().At(0).Resource().Attributes()
		if _, ok := attrs.Get(attrInstanceName); ok {
			t.Errorf("expected no %s attribute when pid attribute is absent", attrInstanceName)
		}
	})
}

func TestIncusAttrProcessor_startup(t *testing.T) {
	t.Run("startup returns without waiting for incus", func(t *testing.T) {
		started := make(chan struct{})
		blockingStart := func(ctx context.Context) error {
			close(started)
			<-ctx.Done()
			return ctx.Err()
		}
		p := newIncusAttrProcessor(nopSettings(), &processorConfig{}, nil, blockingStart)

		done := make(chan error, 1)
		go func() { done <- p.startup(context.Background(), componenttest.NewNopHost()) }()
		select {
		case err := <-done:
			if err != nil {
				t.Fatalf("expected nil, got %v", err)
			}
		case <-time.After(time.Second):
			t.Fatal("startup blocked; expected non-blocking return")
		}
		<-started
		_ = p.shutdown(context.Background())
	})
}

func TestProcessProfiles_resourceAttributes(t *testing.T) {
	t.Run("cache returns empty Location", func(t *testing.T) {
		procRoot := t.TempDir()
		writeCgroup(t, procRoot, "1122", "0::/lxc.payload.bazz\n")
		seed := incus.InstanceInfo{Name: "bazz", Project: "default", Location: ""}
		cacheLookup := metadata.NewCache(nil, &noopEventSource{}, warmupWith(seed), zap.NewNop())
		_ = cacheLookup.Start(t.Context())
		src := metadata.NewSource(cacheLookup, procRoot)
		p := newIncusAttrProcessor(nopSettings(), &processorConfig{}, src, noStart)

		pd, _ := newProfilesWithPID(1122)
		got, err := p.processProfiles(context.Background(), pd)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		attrs := got.ResourceProfiles().At(0).Resource().Attributes()
		if v, ok := attrs.Get(attrInstanceLocation); ok {
			t.Errorf("set %s=%q from an empty cache value; empty values must be skipped",
				attrInstanceLocation, v.Str())
		}
	})

	t.Run("incus.instance.name already set upstream", func(t *testing.T) {
		procRoot := t.TempDir()
		writeCgroup(t, procRoot, "1122", "0::/lxc.payload.bar\n")
		seed := incus.InstanceInfo{Name: "bar", Project: "default", Location: "node-0"}
		cacheLookup := metadata.NewCache(nil, &noopEventSource{}, warmupWith(seed), zap.NewNop())
		_ = cacheLookup.Start(t.Context())
		src := metadata.NewSource(cacheLookup, procRoot)
		p := newIncusAttrProcessor(nopSettings(), &processorConfig{}, src, noStart)

		pd, rp := newProfilesWithPID(1122)
		rp.Resource().Attributes().PutStr(attrInstanceName, "preset")
		got, err := p.processProfiles(context.Background(), pd)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		attrs := got.ResourceProfiles().At(0).Resource().Attributes()
		if v, _ := attrs.Get(attrInstanceName); v.Str() != "preset" {
			t.Errorf("overwrote %s with cache value %q; processor must be additive and preserve upstream values",
				attrInstanceName, v.Str())
		}
	})
}

func assertAttr(t *testing.T, attrs pcommon.Map, key, want string) {
	t.Helper()
	got, ok := attrs.Get(key)
	if !ok {
		t.Errorf("attribute %s missing", key)
		return
	}
	if got.Str() != want {
		t.Errorf("%s: want %q, got %q", key, want, got.Str())
	}
}

func newProfilesWithPID(pid int64) (pprofile.Profiles, pprofile.ResourceProfiles) {
	pd := pprofile.NewProfiles()
	rp := pd.ResourceProfiles().AppendEmpty()
	rp.Resource().Attributes().PutInt(attrPID, pid)
	return pd, rp
}

func nopSettings() processor.Settings {
	return processor.Settings{
		TelemetrySettings: component.TelemetrySettings{Logger: zap.NewNop()},
	}
}

var noStart = func(ctx context.Context) error { return nil }

type noopEventSource struct {
}

func (n *noopEventSource) Subscribe(_ context.Context, _ func(e incus.InstanceEvent), onConn func()) {
	if onConn != nil {
		onConn()
	}
}

var _ metadata.InstanceEvents = &noopEventSource{}
