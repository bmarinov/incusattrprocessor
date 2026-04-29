package incusattrprocessor

import (
	"context"
	"fmt"

	"github.com/bmarinov/otelcol-processor-incus/internal/instanceid"
)

type cgroupMetadataSource struct {
	procRoot string
}

func newCgroupMetadataSource() *cgroupMetadataSource {
	return &cgroupMetadataSource{procRoot: "/proc"}
}

func (s *cgroupMetadataSource) GetInstanceMetadata(_ context.Context, pid string) (InstanceMetadata, error) {
	name, err := instanceid.FromPID(s.procRoot, pid)

	if err != nil {
		return InstanceMetadata{}, fmt.Errorf("pid %s: %w", pid, err)
	}

	return InstanceMetadata{Name: name}, nil
}
