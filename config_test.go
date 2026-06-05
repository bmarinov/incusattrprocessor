package incusattrprocessor

import (
	"testing"

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
	t.Run("missing cert file returns error", func(t *testing.T) {
		h := &httpsConfig{
			URL:        "https://incus.example.com:8443",
			ClientCert: "/nonexistent/client.crt",
			ClientKey:  "/nonexistent/client.key",
			ServerCert: "/nonexistent/server.crt",
		}
		_, err := h.load()
		if err == nil {
			t.Error("expected error for missing cert files")
		}
	})
}
