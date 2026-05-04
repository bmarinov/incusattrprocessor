package incusattrprocessor

import (
	"context"
	"fmt"

	"github.com/bmarinov/otelcol-processor-incus/internal/cgroup"
	"github.com/bmarinov/otelcol-processor-incus/internal/incus"
)

type instanceLookup interface {
	GetInstance(ctx context.Context, project, name string) (incus.InstanceInfo, error)
}

type cgroupMetadataSource struct {
	procRoot string
	client   instanceLookup
}

func newCgroupMetadataSource() *cgroupMetadataSource {
	return &cgroupMetadataSource{procRoot: "/proc"}
}

func (s *cgroupMetadataSource) GetInstanceMetadata(ctx context.Context, pid string) (InstanceMetadata, error) {
	cPath, err := cgroup.Read(s.procRoot, pid)
	if err != nil {
		return InstanceMetadata{}, fmt.Errorf("reading cgroup for pid %s: %w", pid, err)
	}
	label, err := cgroup.ParseLXC(cPath)
	if err != nil {
		return InstanceMetadata{}, err
	}
	project, name := incus.SplitLabel(label)
	instance, err := s.client.GetInstance(ctx, project, name)
	if err != nil {
		return InstanceMetadata{}, fmt.Errorf("incus lookup %s/%s: %w", project, name, err)
	}
	return InstanceMetadata{
		Name:     name,
		Project:  project,
		Location: instance.Location}, nil
}
