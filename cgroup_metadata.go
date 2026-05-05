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
	lookup   instanceLookup
}

func newCgroupMetadataSource(lookup instanceLookup) *cgroupMetadataSource {
	return &cgroupMetadataSource{
		procRoot: "/proc",
		lookup:   lookup,
	}
}

func (s *cgroupMetadataSource) GetInstanceMetadata(ctx context.Context, pid string) (incus.InstanceInfo, error) {
	cPath, err := cgroup.Read(s.procRoot, pid)
	if err != nil {
		return incus.InstanceInfo{}, fmt.Errorf("reading cgroup for pid %s: %w", pid, err)
	}
	label, err := cgroup.ParseLXC(cPath)
	if err != nil {
		return incus.InstanceInfo{}, err
	}
	project, name := incus.SplitLabel(label)
	instance, err := s.lookup.GetInstance(ctx, project, name)
	if err != nil {
		return incus.InstanceInfo{}, fmt.Errorf("incus lookup %s/%s: %w", project, name, err)
	}
	return instance, nil
}
