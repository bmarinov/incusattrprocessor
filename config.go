package incusattrprocessor

import (
	"fmt"
	"os"

	"github.com/bmarinov/otelcol-processor-incus/internal/incus"
	"go.opentelemetry.io/collector/component"
)

type processorConfig struct {
	Connection connectionConfig `mapstructure:"connection"`
}

type connectionConfig struct {
	SocketPath string       `mapstructure:"socket_path"`
	HTTPS      *httpsConfig `mapstructure:"https"`
}

// httpsConfig for mTLS access over HTTPS.
// The cert fields are file paths to PEM files.
type httpsConfig struct {
	URL        string `mapstructure:"url"`
	ClientCert string `mapstructure:"client_cert"`
	ClientKey  string `mapstructure:"client_key"`
	ServerCert string `mapstructure:"server_cert"`
}

func (h *httpsConfig) load() (incus.HTTPSConfig, error) {
	clientCert, err := os.ReadFile(h.ClientCert)
	if err != nil {
		return incus.HTTPSConfig{}, fmt.Errorf("reading client cert %s: %w", h.ClientCert, err)
	}
	clientKey, err := os.ReadFile(h.ClientKey)
	if err != nil {
		return incus.HTTPSConfig{}, fmt.Errorf("reading client key %s: %w", h.ClientKey, err)
	}
	serverCert, err := os.ReadFile(h.ServerCert)
	if err != nil {
		return incus.HTTPSConfig{}, fmt.Errorf("reading server cert %s: %w", h.ServerCert, err)
	}
	return incus.HTTPSConfig{
		URL:        h.URL,
		ClientCert: string(clientCert),
		ClientKey:  string(clientKey),
		ServerCert: string(serverCert),
	}, nil
}

func createDefaultConfig() component.Config {
	return &processorConfig{
		Connection: connectionConfig{
			SocketPath: "",
		},
	}
}
