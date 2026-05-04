package incusattrprocessor

import (
	"context"
	"errors"
	"fmt"

	"github.com/bmarinov/otelcol-processor-incus/internal/cgroup"
	"github.com/bmarinov/otelcol-processor-incus/internal/incus"
	incusclient "github.com/lxc/incus/v6/client"
	"go.opentelemetry.io/collector/component"
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
	ctx context.Context,
	params processor.Settings,
	cfg *processorConfig,
	meta MetadataSource,
) *incusAttrProcessor {
	_, cancel := context.WithCancel(ctx)

	// TODO: incus conn -> new client from local internal/incus package

	return &incusAttrProcessor{
		cancel:   cancel,
		config:   *cfg,
		metadata: meta,
		logger:   params.Logger,
	}
}

type incusAttrProcessor struct {
	cancel   context.CancelFunc
	config   processorConfig
	metadata MetadataSource
	logger   *zap.Logger
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
		meta, err := p.metadata.GetInstanceMetadata(ctx, pid)
		if err != nil {
			if !errors.Is(err, cgroup.ErrNotContainer) {
				p.logger.Debug("metadata lookup failed", zap.String("pid", pid), zap.Error(err))
			}
			continue
		}
		matched++

		p.logger.Debug("matched container", zap.String("pid", pidVal.Str()), zap.String("container", meta.Name))
		attrs.PutStr(attrInstanceName, meta.Name)
		attrs.PutStr(attrInstanceProject, meta.Project)
		attrs.PutStr(attrInstanceLocation, meta.Location)
	}
	if total > 0 {
		p.logger.Debug("batch", zap.Int("matched", matched), zap.Int("total", total))
	}
	return pd, nil
}

func (p *incusAttrProcessor) startup(ctx context.Context, _ component.Host) error {
	src, ok := p.metadata.(*cgroupMetadataSource)
	if !ok {
		return nil
	}
	conn, err := incusclient.ConnectIncusUnixWithContext(ctx, "", nil)
	if err != nil {
		return fmt.Errorf("connecting to incus daemon: %w", err)
	}
	src.client = incus.New(conn)
	return nil
}

func (p *incusAttrProcessor) shutdown(_ context.Context) error {
	p.cancel()
	return nil
}
