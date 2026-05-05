package incusattrprocessor

import (
	"testing"

	"go.opentelemetry.io/collector/confmap/confmaptest"
)

func TestConfig(t *testing.T) {
	t.Run("socket path config from file", func(t *testing.T) {
		cm, err := confmaptest.LoadConf("testdata/config.yaml")
		if err != nil {
			t.Fatal(err)
		}

		expected := "/var/lib/incus/unix.socket"
		subcfg, err := cm.Sub("processors::incusattr")
		if err != nil {
			t.Fatal(err)
		}
		cfg := createDefaultConfig()
		err = subcfg.Unmarshal(cfg)
		if err != nil {
			t.Fatal(err)
		}

		processorConfig := cfg.(*processorConfig)
		if processorConfig.Connection.SocketPath != expected {
			t.Errorf("expected %q got %q", expected, processorConfig.Connection.SocketPath)
		}
	})

}
