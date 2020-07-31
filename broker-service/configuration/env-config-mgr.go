package configuration

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
)

// const (
// 	BASE_BROKER_NAME = "broker"
// )

// EnvironmentConfigManager gets configurations from environment variables
type EnvironmentConfigManager struct {
}

// GetAdapterConfigurations gets the configuration from environment variables
func (e EnvironmentConfigManager) GetAdapterConfigurations(ctx context.Context) []BrokerConfiguration {

	configs := make([]BrokerConfiguration, 0)

	for i := 1; i < 101; i++ {
		prefix := fmt.Sprintf("BROKER%d_", i)

		variables := e.getAllConfigValues(prefix)

		name, ok := variables["NAME"]
		if !ok || name == "" {
			continue
		}

		config := BrokerConfiguration{
			Name: variables["NAME"],
			Type: variables["TYPE"],
			URL:  variables["URL"],
			All:  variables,
			User: variables["USER"],
			Pass: variables["PASS"],
		}

		configs = append(configs, config)
	}

	e.printConfigs(ctx, configs)
	return configs
}

func (e EnvironmentConfigManager) getAllConfigValues(prefix string) map[string]string {

	variablesWithPrefix := make(map[string]string)

	for _, element := range os.Environ() {
		variable := strings.Split(element, "=")
		if strings.Contains(variable[0], prefix) {
			key := strings.Replace(variable[0], prefix, "", 1)
			variablesWithPrefix[key] = variable[1]
		}
	}
	return variablesWithPrefix
}

func (e EnvironmentConfigManager) printConfigs(ctx context.Context, configs []BrokerConfiguration) {
	log.Println("Brokers to build...")
	for _, config := range configs {
		log.Printf("Configuration:\nName: %s\nUrl: %s\nUser: %s\nType: %s\n\n",
			config.Name, config.URL, config.User, config.Type)
	}
}
