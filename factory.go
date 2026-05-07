package incusattrprocessor

import (
	"context"

	"github.com/bmarinov/otelcol-processor-incus/internal/incus"
	"github.com/bmarinov/otelcol-processor-incus/internal/metadata"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/xconsumer"
	"go.opentelemetry.io/collector/processor"
	"go.opentelemetry.io/collector/processor/processorhelper/xprocessorhelper"
	"go.opentelemetry.io/collector/processor/xprocessor"
)

// typeStr defines the unique type identifier for the processor.
var typeStr = component.MustNewType("incusattr")

const defaultProcRoot = "/proc"

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

	incusClient := incus.New(pConfig.Connection.SocketPath, params.Logger)
	cache := metadata.NewCache(incusClient,
		func(ctx context.Context) ([]incus.InstanceInfo, error) {
			return incusClient.GetAllInstances(ctx)
		},
		params.Logger,
	)
	lookup := metadata.NewSource(cache, defaultProcRoot)

	p := newIncusAttrProcessor(params, pConfig, lookup, func(ctx context.Context) error {
		err := incusClient.Start(ctx)
		if err != nil {
			return err
		}
		return cache.Start(ctx)
	})

	consumerCapabilities := consumer.Capabilities{MutatesData: true}
	processor, err := xprocessorhelper.NewProfiles(ctx, params, cfg, nextProfilesConsumer,
		p.processProfiles,
		xprocessorhelper.WithCapabilities(consumerCapabilities),
		xprocessorhelper.WithStart(p.startup),
		xprocessorhelper.WithShutdown(p.shutdown),
	)

	return processor, err
}
