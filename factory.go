package incusattrprocessor

import (
	"context"

	"github.com/bmarinov/otelcol-processor-incus/internal/incus"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/xconsumer"
	"go.opentelemetry.io/collector/processor"
	"go.opentelemetry.io/collector/processor/processorhelper/xprocessorhelper"
	"go.opentelemetry.io/collector/processor/xprocessor"
)

// typeStr defines the unique type identifier for the processor.
var typeStr = component.MustNewType("incusattr")

func NewFactory() processor.Factory {
	return xprocessor.NewFactory(
		typeStr,
		createDefaultConfig,
		xprocessor.WithProfiles(createProfilesProcessor, component.StabilityLevelAlpha),
	)
}

func createProfilesProcessor(
	ctx context.Context,
	params processor.Settings,
	cfg component.Config,
	nextProfilesConsumer xconsumer.Profiles,
) (xprocessor.Profiles, error) {
	pConfig := cfg.(*processorConfig)

	// todo: middleware/cache:
	incusClient := incus.New(pConfig.Connection.SocketPath)
	lookup := newCgroupMetadataSource(incusClient)

	p := newIncusAttrProcessor(params, pConfig, lookup, incusClient.Start)

	consumerCapabilities := consumer.Capabilities{MutatesData: true}
	processor, err := xprocessorhelper.NewProfiles(ctx, params, cfg, nextProfilesConsumer,
		p.processProfiles,
		xprocessorhelper.WithCapabilities(consumerCapabilities),
		xprocessorhelper.WithStart(p.startup),
		xprocessorhelper.WithShutdown(p.shutdown),
	)

	return processor, err
}
