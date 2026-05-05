package incusattrprocessor

import (
	"context"
	"testing"

	"github.com/bmarinov/otelcol-processor-incus/internal/incus"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pprofile"
	"go.opentelemetry.io/collector/processor"
	"go.uber.org/zap"
)

// no op start func
var nop = func(ctx context.Context) error { return nil }

func TestIncusAttrProcessor_processProfiles(t *testing.T) {
	t.Run("adds container metadata to resource when pid matches a running instance", func(t *testing.T) {
		procRoot := t.TempDir()
		writeCgroup(t, procRoot, "1122", "0::/lxc.payload.container-foo\n")
		src := &cgroupMetadataSource{
			procRoot: procRoot,
			lookup:   &fakeInstanceLookup{info: incus.InstanceInfo{Location: "node-0"}},
		}
		p := newIncusAttrProcessor(
			nopSettings(),
			&processorConfig{},
			src,
			nop)

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
		src := &cgroupMetadataSource{
			procRoot: procRoot,
			lookup:   &fakeInstanceLookup{},
		}
		p := newIncusAttrProcessor(nopSettings(), &processorConfig{}, src, nop)

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
		src := &cgroupMetadataSource{
			procRoot: t.TempDir(),
			lookup:   &fakeInstanceLookup{},
		}
		p := newIncusAttrProcessor(nopSettings(), &processorConfig{}, src, nop)

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
