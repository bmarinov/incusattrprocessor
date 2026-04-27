package incusattrprocessor

import (
	"context"

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
	pCfg := cfg.(*processorConfig)
	// TODO: pass actual incus meta source
	p := newIncusAttrProcessor(ctx, pCfg, nil)

	consumerCapabilities := consumer.Capabilities{MutatesData: true}
	foo, err := xprocessorhelper.NewProfiles(ctx, params, cfg, nextProfilesConsumer,
		p.processProfiles,
		xprocessorhelper.WithCapabilities(consumerCapabilities),
		xprocessorhelper.WithStart(p.startup),
		xprocessorhelper.WithShutdown(p.shutdown),
	)

	return foo, err
}
