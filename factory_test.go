package incusattrprocessor

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/consumer/xconsumer"
	"go.opentelemetry.io/collector/processor/xprocessor"
)

func TestFactory_DefaultConfigIsValid(t *testing.T) {
	err := componenttest.
		CheckConfigStruct(createDefaultConfig())
	if err != nil {
		t.Fatal(err)
	}
}

func TestFactory_Profiles(t *testing.T) {
	t.Run("declares that it mutates data", func(t *testing.T) {
		p := newFactoryProfiles(t, new(consumertest.ProfilesSink))
		if !p.Capabilities().MutatesData {
			t.Error("expected MutatesData to be true")
		}
	})

	t.Run("forwards profiles to the next consumer", func(t *testing.T) {
		sink := new(consumertest.ProfilesSink)
		p := newFactoryProfiles(t, sink)
		if err := p.Start(t.Context(), componenttest.NewNopHost()); err != nil {
			t.Fatalf("start returned unexpected error: %v", err)
		}
		t.Cleanup(func() { _ = p.Shutdown(context.Background()) })

		pd, _ := newProfilesWithPID(1122)
		if err := p.ConsumeProfiles(t.Context(), pd); err != nil {
			t.Fatalf("consume returned unexpected error: %v", err)
		}

		got := sink.AllProfiles()
		if len(got) != 1 {
			t.Fatalf("sink batches: expected 1, got %d", len(got))
		}
		if n := got[0].ResourceProfiles().Len(); n != 1 {
			t.Errorf("resource profiles: expected 1, got %d", n)
		}
	})

	t.Run("starts without waiting for an unreachable incus", func(t *testing.T) {
		p := newFactoryProfiles(t, new(consumertest.ProfilesSink))

		done := make(chan error, 1)
		go func() { done <- p.Start(t.Context(), componenttest.NewNopHost()) }()
		select {
		case err := <-done:
			if err != nil {
				t.Fatalf("start returned unexpected error: %v", err)
			}
		case <-time.After(time.Second):
			t.Fatal("start blocked; expected non-blocking return")
		}

		if err := p.Shutdown(context.Background()); err != nil {
			t.Fatalf("shutdown returned unexpected error: %v", err)
		}
	})

	t.Run("fails when the https certs cannot be read", func(t *testing.T) {
		cfg := createDefaultConfig().(*processorConfig)
		cfg.Connection.HTTPS = &httpsConfig{
			URL:        "https://incus.example.com:8443",
			ClientCert: filepath.Join(t.TempDir(), "missing.crt"),
		}
		_, err := profilesFactory(t).
			CreateProfiles(t.Context(), nopSettings(), cfg, new(consumertest.ProfilesSink))
		if err == nil {
			t.Error("expected error for an unreadable client cert")
		}
	})
}

func profilesFactory(t *testing.T) xprocessor.Factory {
	t.Helper()
	f := NewFactory()
	pf, ok := f.(xprocessor.Factory)
	if !ok {
		t.Fatalf("NewFactory: expected an xprocessor.Factory, got %T", f)
	}
	return pf
}

func newFactoryProfiles(t *testing.T, next xconsumer.Profiles) xprocessor.Profiles {
	t.Helper()
	cfg := createDefaultConfig().(*processorConfig)
	// an empty SocketPath resolves to $INCUS_SOCKET or the system default
	cfg.Connection.SocketPath = filepath.Join(t.TempDir(), "missing.socket")

	p, err := profilesFactory(t).CreateProfiles(t.Context(), nopSettings(), cfg, next)
	if err != nil {
		t.Fatalf("CreateProfiles returned unexpected error: %v", err)
	}
	return p
}
