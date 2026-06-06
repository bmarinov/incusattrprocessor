package incusattrprocessor

import (
	"testing"

	"go.opentelemetry.io/collector/component/componenttest"
)

func TestFactory_DefaultConfigIsValid(t *testing.T) {
	err := componenttest.
		CheckConfigStruct(createDefaultConfig())
	if err != nil {
		t.Fatal(err)
	}
}
