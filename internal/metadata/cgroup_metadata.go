package metadata

import (
	"context"
	"fmt"
	"slices"
	"sync"
	"time"

	"github.com/bmarinov/otelcol-processor-incus/internal/cgroup"
	"github.com/bmarinov/otelcol-processor-incus/internal/incus"
	"go.uber.org/zap"
)

type InstanceLookup interface {
	GetInstance(ctx context.Context, project, name string) (incus.InstanceInfo, error)
}

type InstanceEvents interface {
	Subscribe(ctx context.Context, callback func(e incus.InstanceEvent))
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
	events       InstanceEvents
	mu           sync.RWMutex
	instanceMeta map[instanceKey]incus.InstanceInfo
	warmup       WarmupFunc
	log          *zap.Logger
}

type WarmupFunc func(ctx context.Context) ([]incus.InstanceInfo, error)

func NewCache(lookup InstanceLookup,
	events InstanceEvents,
	w WarmupFunc,
	log *zap.Logger) *Cache {
	return &Cache{
		lookup:       lookup,
		events:       events,
		instanceMeta: map[instanceKey]incus.InstanceInfo{},
		warmup:       w,
		log:          log,
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
		// TODO: decoupling from caller context, reasses approach & timeout:
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
	err := doWarmup(ctx, c)
	if err != nil {
		return fmt.Errorf("client warmup: %w", err)
	}

	c.events.Subscribe(ctx, func(e incus.InstanceEvent) {
		c.mu.RLock()
		_, inCache := c.instanceMeta[instanceKey{project: e.Project, name: e.Name}]
		c.mu.RUnlock()
		if !inCache {
			return
		}

		c.mu.Lock()
		_, inCache = c.instanceMeta[instanceKey{project: e.Project, name: e.Name}]

		// TODO: use a map:
		if slices.Contains(incus.EventsPurgeCache, e.Action) && inCache {
			// TODO: see if old_name can be exposed in a clean way
			if e.Action != "instance-renamed" {
				delete(c.instanceMeta, instanceKey{project: e.Project, name: e.Name})
			}
		}

		// need to unlock so GetInstance can acquire a R -> W lock
		c.mu.Unlock()

		if slices.Contains(incus.EventsUpdateCache, e.Action) {
			_, err := c.GetInstance(ctx, e.Project, e.Name)
			if err != nil {
				// TODO: warn? cache never returns err
			}
		}
	})

	return nil
}

func doWarmup(ctx context.Context, c *Cache) error {
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
