package owspanprocessor

import "go.opentelemetry.io/collector/config/configmodels"

type Config struct {
	configmodels.ProcessorSettings `mapstructure:",squash"`
	// Address of OpenWhisk API (e.g.: localhost:31001)
	OwHost string `mapstructure:"ow_host"`
	// E.g.: 23bc46b1-71f6-4ed5-8c54-816aa4f8c502:123zO3xZCLrMN6v2BKK1dXYFpXlPkccOFqm12CdAsMgRU4VrNZ9lyGVCGuMDGIwP
	OwAuthToken string `mapstructure:"ow_auth_token"`
	// Enable logging for processor (default false)
	Logging bool `mapstructure:"logging"`
}
