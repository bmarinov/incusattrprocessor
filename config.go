package incusattrprocessor

import "go.opentelemetry.io/collector/component"

type processorConfig struct {
	Connection connectionConfig `mapstructure:"connection"`
}

type connectionConfig struct {
	SocketPath string `mapstructure:"socket_path"`
}

func createDefaultConfig() component.Config {
	return &processorConfig{
		Connection: connectionConfig{
			SocketPath: "",
		},
	}
}
