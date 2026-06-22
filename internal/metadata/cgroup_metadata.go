package metadata

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/bmarinov/incusattrprocessor/internal/cgroup"
	"github.com/bmarinov/incusattrprocessor/internal/incus"
	"go.uber.org/zap"
)

type cacheAction int

const (
	actionPurge cacheAction = iota
	actionUpdate
)

// instance to cache action map.
var instanceActions = map[string]cacheAction{
	incus.EventInstanceStopped:   actionPurge,
	incus.EventInstanceShutdown:  actionPurge,
	incus.EventInstanceDeleted:   actionPurge,
	incus.EventInstanceRenamed:   actionUpdate,
	incus.EventInstanceStarted:   actionUpdate,
	incus.EventInstanceRestarted: actionUpdate,
}

type InstanceLookup interface {
	GetInstance(ctx context.Context, project, name string) (incus.InstanceInfo, error)
}

type InstanceEvents interface {
	//Subscribe registers a callback for instance events.
	// onConnect is called each time the event stream is established (including initial connect).
	Subscribe(ctx context.Context,
		callback func(e incus.InstanceEvent),
		onConnect func(),
	)
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
	ready := make(chan error, 1)
	var once sync.Once

	c.events.Subscribe(ctx,
		func(e incus.InstanceEvent) {
			action, got := instanceActions[e.Action]
			if !got {
				return
			}

			c.mu.Lock()
			if e.Action == incus.EventInstanceRenamed {
				delete(c.instanceMeta, instanceKey{project: e.Project, name: e.OldName})
			} else {
				delete(c.instanceMeta, instanceKey{project: e.Project, name: e.Name})
			}
			c.mu.Unlock()
			if action == actionUpdate {
				// c.mu has to be released so GetInstance can acquire the lock.
				_, _ = c.GetInstance(ctx, e.Project, e.Name)
			}
		},
		func() {
			err := doWarmup(ctx, c)
			if err != nil {
				c.log.Debug("cache warmup on connect failed", zap.Error(err))
			}
			once.Do(func() { ready <- err })
		},
	)

	select {
	case err := <-ready:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

func doWarmup(ctx context.Context, c *Cache) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	instances, err := c.warmup(ctx)
	if err != nil {
		return fmt.Errorf("cache warmup: %w", err)
	}

	c.instanceMeta = make(map[instanceKey]incus.InstanceInfo, len(instances))
	for _, v := range instances {
		c.instanceMeta[instanceKey{project: v.Project, name: v.Name}] = v
	}
	return nil
}

var _ InstanceLookup = &Cache{}
