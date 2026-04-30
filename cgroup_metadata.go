package incusattrprocessor

import (
	"context"
	"fmt"

	"github.com/bmarinov/otelcol-processor-incus/internal/cgroup"
)

type cgroupMetadataSource struct {
	procRoot string
}

func newCgroupMetadataSource() *cgroupMetadataSource {
	return &cgroupMetadataSource{procRoot: "/proc"}
}

func (s *cgroupMetadataSource) GetInstanceMetadata(_ context.Context, pid string) (InstanceMetadata, error) {
	cPath, err := cgroup.Read(s.procRoot, pid)
	if err != nil {
		return InstanceMetadata{}, fmt.Errorf("reading cgroup file: %w", err)
	}
	instance, err := cgroup.ParseLXC(cPath)
	if err != nil {
		return InstanceMetadata{}, err
	}

	if err != nil {
		return InstanceMetadata{}, fmt.Errorf("pid %s: %w", pid, err)
	}

	// todo: split project_instancename
	return InstanceMetadata{Name: instance}, nil
}
