package owtraceprocessor

import (
	"context"
	"errors"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configmodels"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/processor/processorhelper"
)

const (
	// The value of "type" key in configuration.
	typeStr = "owtraceprocessor"
	defaultLogging bool = false 
)

// NewFactory creates a factory for the routing processor.
func NewFactory() component.ProcessorFactory {
	return processorhelper.NewFactory(
		typeStr,
		createDefaultConfig,
		processorhelper.WithTraces(createTraceProcessor),
	)
}

func createDefaultConfig() configmodels.Processor {
	return &Config{
		ProcessorSettings: configmodels.ProcessorSettings{
			TypeVal: typeStr,
			NameVal: typeStr,
		},
		Logging: defaultLogging,
	}
}

func createTraceProcessor(
	_ context.Context,
	params component.ProcessorCreateParams,
	cfg configmodels.Processor,
	nextConsumer consumer.TracesConsumer,
) (component.TracesProcessor, error) {
	config := cfg.(*Config)
	if config.OwHost == "" {
		return nil, errors.New("processor config requires a non-empty 'ow_host'")
	}
	if config.OwAuthToken == "" {
		return nil, errors.New("processor config requires a non-empty 'ow_auth_token'")
	}
	return newOwTraceProcessor(nextConsumer, config, params.Logger)
}