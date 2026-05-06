package metadata

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/bmarinov/otelcol-processor-incus/internal/cgroup"
	"github.com/bmarinov/otelcol-processor-incus/internal/incus"
	"go.uber.org/zap"
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

type Cache struct {
	lookup       InstanceLookup
	mu           sync.RWMutex
	instanceMeta map[instanceKey]incus.InstanceInfo
	warmup       WarmupFunc
	log          *zap.Logger
}

type WarmupFunc func(ctx context.Context) ([]incus.InstanceInfo, error)

func NewCache(lookup InstanceLookup,
	w WarmupFunc,
	log *zap.Logger) *Cache {
	return &Cache{
		lookup:       lookup,
		instanceMeta: map[instanceKey]incus.InstanceInfo{},
		warmup:       w,
	}
}

func (c *Cache) GetInstance(ctx context.Context, project string, name string) (incus.InstanceInfo, error) {
	c.mu.RLock()
	entry, got := c.instanceMeta[instanceKey{project: project, name: name}]
	c.mu.RUnlock()

	if got {
		return entry, nil
	}

	go func() {
		ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
		defer cancel()
		i, err := c.lookup.GetInstance(ctx, project, name)
		if err != nil {
			c.log.Debug("background instance fetch", zap.Error(err), zap.String("incus.project", project), zap.String("incus.instance", name))
			return
		}
		c.mu.Lock()
		defer c.mu.Unlock()
		c.instanceMeta[instanceKey{project: project, name: name}] = i
	}()

	return incus.InstanceInfo{
		Name:    name,
		Project: project,
	}, nil
}

func (c *Cache) Start(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

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
