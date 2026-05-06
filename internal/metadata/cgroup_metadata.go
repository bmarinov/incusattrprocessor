package metadata

import (
	"context"
	"fmt"

	"github.com/bmarinov/otelcol-processor-incus/internal/cgroup"
	"github.com/bmarinov/otelcol-processor-incus/internal/incus"
)

type InstanceLookup interface {
	GetInstance(ctx context.Context, project, name string) (incus.InstanceInfo, error)
}

type CgroupMetadataSource struct {
	procRoot string
	lookup   InstanceLookup
}

func NewSource(lookup InstanceLookup, procRoot string) *CgroupMetadataSource {
	return &CgroupMetadataSource{
		procRoot: procRoot,
		lookup:   lookup,
	}
}

func (s *CgroupMetadataSource) GetInstanceMetadata(ctx context.Context, pid string) (incus.InstanceInfo, error) {
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

type instanceKey struct {
	project string
	name    string
}

func NewCache(lookup InstanceLookup, w WarmupFunc) *Cache {
	return &Cache{
		lookup:       lookup,
		instanceMeta: map[instanceKey]incus.InstanceInfo{},
		warmup:       w,
	}
}

type Cache struct {
	lookup       InstanceLookup
	instanceMeta map[instanceKey]incus.InstanceInfo
	warmup       WarmupFunc
}

type WarmupFunc func(ctx context.Context) ([]incus.InstanceInfo, error)

func (c *Cache) GetInstance(ctx context.Context, project string, name string) (incus.InstanceInfo, error) {
	entry, got := c.instanceMeta[instanceKey{project: project, name: name}]
	if !got {
		return incus.InstanceInfo{
			Name:    name,
			Project: project,
		}, nil
	}
	return entry, nil
}

func (c *Cache) Start(ctx context.Context) error {
	instances, err := c.warmup(ctx)

	if err != nil {
		return fmt.Errorf("cache warmup: %w", err)
	}

	for _, v := range instances {
		c.instanceMeta[instanceKey{project: v.Project, name: v.Name}] = v
	}
	return nil
}

var _ InstanceLookup = &Cache{}
