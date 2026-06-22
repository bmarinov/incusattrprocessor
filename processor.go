package incusattrprocessor

import (
	"context"
	"errors"

	"github.com/bmarinov/incusattrprocessor/internal/cgroup"
	"github.com/bmarinov/incusattrprocessor/internal/incus"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pprofile"
	"go.opentelemetry.io/collector/processor"
	"go.uber.org/zap"
)

const (
	attrInstanceName     = "incus.instance.name"
	attrInstanceProject  = "incus.instance.project"
	attrInstanceLocation = "incus.instance.location"
)

const attrPID = "process.pid"

type MetadataSource interface {
	GetInstanceMetadata(ctx context.Context, id string) (incus.InstanceInfo, error)
}

func newIncusAttrProcessor(
	params processor.Settings,
	cfg *processorConfig,
	meta MetadataSource,
	startFn func(context.Context) error,
) *incusAttrProcessor {
	return &incusAttrProcessor{
		config: *cfg,
		lookup: meta,
		logger: params.Logger,
		start:  startFn,
	}
}

type incusAttrProcessor struct {
	cancel context.CancelFunc
	config processorConfig
	lookup MetadataSource
	logger *zap.Logger
	start  func(ctx context.Context) error
}

func (p *incusAttrProcessor) processProfiles(ctx context.Context, pd pprofile.Profiles) (pprofile.Profiles, error) {
	total, matched := 0, 0
	for _, rp := range pd.ResourceProfiles().All() {
		attrs := rp.Resource().Attributes()

		pidVal, ok := attrs.Get(attrPID)
		if !ok {
			continue
		}
		total++

		pid := pidVal.AsString()
		meta, err := p.lookup.GetInstanceMetadata(ctx, pid)
		if err != nil {
			if !errors.Is(err, cgroup.ErrNotContainer) {
				p.logger.Debug("metadata lookup failed", zap.String("pid", pid), zap.Error(err))
			}
			continue
		}
		matched++

		p.logger.Debug("matched container", zap.String("pid", pid), zap.String("container", meta.Name))

		setResourceAttr(attrs, attrInstanceName, meta.Name)
		setResourceAttr(attrs, attrInstanceProject, meta.Project)
		setResourceAttr(attrs, attrInstanceLocation, meta.Location)
	}
	if total > 0 {
		p.logger.Debug("batch", zap.Int("matched", matched), zap.Int("total", total))
	}
	return pd, nil
}

func setResourceAttr(attrs pcommon.Map, key, val string) {
	if val == "" {
		return
	}
	if _, ok := attrs.Get(key); ok {
		return
	}
	attrs.PutStr(key, val)
}

func (p *incusAttrProcessor) startup(ctx context.Context, _ component.Host) error {
	background, cancel := context.WithCancel(context.WithoutCancel(ctx))
	p.cancel = cancel

	go func() {
		if err := p.start(background); err != nil {
			p.logger.Warn("incus processor startup", zap.Error(err))
		}
	}()
	return nil
}

func (p *incusAttrProcessor) shutdown(_ context.Context) error {
	if p.cancel != nil {
		p.cancel()
	}
	return nil
}
