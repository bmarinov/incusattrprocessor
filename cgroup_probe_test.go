package incusattrprocessor

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/bmarinov/otelcol-processor-incus/internal/incus"
	"github.com/bmarinov/otelcol-processor-incus/internal/metadata"
	"go.uber.org/zap"
)

// Run on the Incus host:
//
// PID=1234 go test -tags probe -v -run TestProbe_cgroupMetadata
func TestProbe_cgroupMetadata(t *testing.T) {
	t.SkipNow()
	pid := os.Getenv("PID")
	if pid == "" {
		pid = "1"
	}

	incusClient := incus.New("")
	cache := metadata.NewCache(incusClient,
		func(ctx context.Context) ([]incus.InstanceInfo, error) {
			return incusClient.GetAllInstances(ctx)
		},
		zap.Must(zap.NewDevelopment()),
	)
	src := metadata.NewSource(cache, defaultProcRoot)

	meta, err := src.GetInstanceMetadata(context.Background(), pid)
	if err != nil {
		t.Fatalf("GetInstanceMetadata(%s): %v", pid, err)
	}
	fmt.Printf("pid=%s name=%s project=%s location=%s\n",
		pid, meta.Name, meta.Project, meta.Location)
}
