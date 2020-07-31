package configuration

import "context"

type ConfigurationManager interface {
	GetAdapterConfigurations(ctx context.Context) []BrokerConfiguration
}
