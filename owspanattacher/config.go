package owspanattacher

import "go.opentelemetry.io/collector/config/configmodels"

type Config struct {
	configmodels.ProcessorSettings `mapstructure:",squash"`
	// Enable logging for processor (default false)
	Logging bool `mapstructure:"logging"`
}
