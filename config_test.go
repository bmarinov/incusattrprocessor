package incusattrprocessor

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/bmarinov/incusattrprocessor/internal/incus"
	"go.opentelemetry.io/collector/confmap/confmaptest"
)

func loadProcessorConfig(t *testing.T, file string) *processorConfig {
	t.Helper()
	cm, err := confmaptest.LoadConf(file)
	if err != nil {
		t.Fatal(err)
	}
	subcfg, err := cm.Sub("processors::incusattr")
	if err != nil {
		t.Fatal(err)
	}
	cfg := createDefaultConfig()
	if err := subcfg.Unmarshal(cfg); err != nil {
		t.Fatal(err)
	}
	return cfg.(*processorConfig)
}

func TestConfig(t *testing.T) {
	t.Run("unix socket", func(t *testing.T) {
		cfg := loadProcessorConfig(t, "testdata/config.yaml")
		if cfg.Connection.SocketPath != "/var/lib/incus/unix.socket" {
			t.Errorf("SocketPath: got %q", cfg.Connection.SocketPath)
		}
		if cfg.Connection.HTTPS != nil {
			t.Errorf("HTTPS: expected nil when config block is absent, got %+v", cfg.Connection.HTTPS)
		}
	})

	t.Run("https with cert files", func(t *testing.T) {
		cfg := loadProcessorConfig(t, "testdata/config_https.yaml")
		h := cfg.Connection.HTTPS
		if h.URL != "https://incus.example.com:8443" {
			t.Errorf("URL: got %q", h.URL)
		}
		if h.ClientCert != "/etc/incus/client.crt" {
			t.Errorf("ClientCert: got %q", h.ClientCert)
		}
		if h.ClientKey != "/etc/incus/client.key" {
			t.Errorf("ClientKey: got %q", h.ClientKey)
		}
		if h.ServerCert != "/etc/incus/server.crt" {
			t.Errorf("ServerCert: got %q", h.ServerCert)
		}
	})
}

func TestHTTPSConfig_load(t *testing.T) {
	const url = "https://incus.example.com:8443"

	t.Run("all files present", func(t *testing.T) {
		dir := t.TempDir()
		cc := "foo-client-cert"
		ck := "bar-client-key"
		sc := "server-cert"

		h := &httpsConfig{
			URL:        url,
			ClientCert: writeTempFile(t, dir, "client.crt", cc),
			ClientKey:  writeTempFile(t, dir, "client.key", ck),
			ServerCert: writeTempFile(t, dir, "server.crt", sc),
		}
		got, err := h.load()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		expected := incus.HTTPSConfig{
			URL:        url,
			ClientCert: cc,
			ClientKey:  ck,
			ServerCert: sc,
		}
		if got != expected {
			t.Errorf("expected %+v, got %+v", expected, got)
		}
	})

	dir := t.TempDir()
	clientCert := writeTempFile(t, dir, "client.crt", "foo")
	clientKey := writeTempFile(t, dir, "client.key", "bar")
	serverCert := writeTempFile(t, dir, "server.crt", "baz")
	missing := filepath.Join(dir, "missing.pem")

	tests := []struct {
		name       string
		clientCert string
		clientKey  string
		serverCert string
	}{
		{"client cert missing", missing, clientKey, serverCert},
		{"client key missing", clientCert, missing, serverCert},
		{"server cert missing", clientCert, clientKey, missing},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := &httpsConfig{URL: url, ClientCert: tc.clientCert, ClientKey: tc.clientKey, ServerCert: tc.serverCert}
			_, err := h.load()
			if err == nil {
				t.Error("expected error for missing file")
			}
		})
	}
}

func writeTempFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	return path
}
