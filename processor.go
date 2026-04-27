package incusattrprocessor

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/pprofile"
)

const (
	attrInstanceName     = "incus.instance.name"
	attrInstanceProject  = "incus.instance.project"
	attrInstanceLocation = "incus.instance.location"
)

const attrPID = "process.pid"

type MetadataSource interface {
	GetInstanceMetadata(ctx context.Context, id string) (InstanceMetadata, error)
}

type InstanceMetadata struct {
	Name     string
	Project  string
	Location string
}

func newIncusAttrProcessor(ctx context.Context, cfg *processorConfig, meta MetadataSource) *incusAttrProcessor {
	_, cancel := context.WithCancel(ctx)
	return &incusAttrProcessor{
		cancel:   cancel,
		config:   *cfg,
		metadata: meta,
	}
}

type incusAttrProcessor struct {
	cancel   context.CancelFunc
	config   processorConfig
	metadata MetadataSource
}

func (p *incusAttrProcessor) processProfiles(ctx context.Context, pd pprofile.Profiles) (pprofile.Profiles, error) {
	for _, rp := range pd.ResourceProfiles().All() {
		attrs := rp.Resource().Attributes()

		pidVal, ok := attrs.Get(attrPID)
		if !ok {
			continue
		}

		meta, err := p.metadata.GetInstanceMetadata(ctx, pidVal.Str())
		if err != nil {
			continue
		}

		attrs.PutStr(attrInstanceName, meta.Name)
		attrs.PutStr(attrInstanceProject, meta.Project)
		attrs.PutStr(attrInstanceLocation, meta.Location)
	}
	return pd, nil
}

func (p *incusAttrProcessor) startup(ctx context.Context, h component.Host) error {
	return nil
}

func (p *incusAttrProcessor) shutdown(_ context.Context) error {
	p.cancel()
	return nil
}
