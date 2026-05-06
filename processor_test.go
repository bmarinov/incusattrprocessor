package incusattrprocessor

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/bmarinov/otelcol-processor-incus/internal/incus"
	"github.com/bmarinov/otelcol-processor-incus/internal/metadata"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pprofile"
	"go.opentelemetry.io/collector/processor"
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
		cacheLookup := metadata.NewCache(nil, warmupWith(seed), zap.NewNop())
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

		cacheLookup := metadata.NewCache(nil, warmupWith(), zap.NewNop())
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
		src := metadata.NewSource(metadata.NewCache(nil, warmupWith(), zap.NewNop()), t.TempDir())
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
		src := metadata.NewSource(metadata.NewCache(nil, warmupWith(), zap.NewNop()), t.TempDir())
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
