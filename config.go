package incusattrprocessor

import "go.opentelemetry.io/collector/component"

type processorConfig struct{}

func createDefaultConfig() component.Config {
	return &processorConfig{}
}
