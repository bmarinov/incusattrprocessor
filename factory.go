package incusattrprocessor

import (
	"context"
	"fmt"
	"time"

	"github.com/bmarinov/incusattrprocessor/internal/incus"
	"github.com/bmarinov/incusattrprocessor/internal/metadata"
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

	var incusClient *incus.Client
	if pConfig.Connection.HTTPS != nil {
		httpsCfg, err := pConfig.Connection.HTTPS.load()
		if err != nil {
			return nil, fmt.Errorf("loading HTTPS config: %w", err)
		}
		incusClient = incus.NewHTTPS(httpsCfg, params.Logger, 3, time.Second)
	} else {
		incusClient = incus.New(pConfig.Connection.SocketPath, params.Logger, 3, time.Second)
	}

	cache := metadata.NewCache(
		incusClient,
		incusClient,
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
