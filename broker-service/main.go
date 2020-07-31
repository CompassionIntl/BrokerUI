package main

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/labstack/echo/middleware"

	"gitlab.com/ciorg/bridge/brokerUI/broker-service/configuration"

	"github.com/labstack/echo"
	"gitlab.com/ciorg/bridge/brokerUI/broker-service/adapters"
	"gitlab.com/ciorg/bridge/brokerUI/broker-service/pkg/service"
)

func main() {

	var configMgr configuration.ConfigurationManager
	configMgr = configuration.EnvironmentConfigManager{}
	configs := configMgr.GetAdapterConfigurations(context.Background())

	mapBrokerNameToAdapter := buildAdapters(configs)
	mapBrokerNameToAdapter["test"] = &adapters.MockAdapter{}

	brokerAdapterManager := service.BrokerAdapterManager{
		MapBrokerNameToAdapter: mapBrokerNameToAdapter,
	}

	e := echo.New()

	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{})) // TODO lock this down to our domain(s)

	setupRestEndpoints(e, brokerAdapterManager)

	err := e.Start(":1355")
	if err != nil {
		e.Logger.Panic("Echo failed to start!", err)
	}

}

func buildAdapters(configs []configuration.BrokerConfiguration) map[string]adapters.Adapter {

	newAdapters := make(map[string]adapters.Adapter)

	for _, config := range configs {
		switch config.Type {
		case "amq":
			amq := getActiveMQAdapter(config)
			if amq != nil {
				newAdapters[config.Name] = amq
			}
		case "rabbitmq":
			rabbit := getRabbitMQAdapter(config)
			if rabbit != nil {
				newAdapters[config.Name] = rabbit
			}
		case "sqs":
			sqsAdapter := getSQSAdapter(config)
			if sqsAdapter != nil {
				newAdapters[config.Name] = sqsAdapter
			}
		default:
			fmt.Printf("Broker type not supported: %s", config.Type)
			continue
		}
	}

	return newAdapters
}

func getActiveMQAdapter(config configuration.BrokerConfiguration) *adapters.ActiveMQAdapter {

	var useTls bool

	useTls = strings.Contains(config.URL, "amqps:")

	consoleUrl := config.All["CONSOLE_URL"]
	consolePass := config.All["CONSOLE_PASS"]
	consoleUser := config.All["CONSOLE_USER"]

	adapter, err := adapters.NewActiveMQAdapter(context.Background(), config.URL, config.User, config.Pass, consoleUrl, consoleUser, consolePass, useTls)
	if err != nil {
		log.Printf("!!Adapter Error!! - %s", err)
		return nil
	}

	return adapter
}

func getRabbitMQAdapter(config configuration.BrokerConfiguration) *adapters.RabbitMQAdapter {

	adapter, err := adapters.NewRabbitMQAdapter(context.Background(), config.URL, config.All["CONSOLE_URL"], config.User, config.Pass, config.All["HOST"])
	if err != nil {
		log.Printf("!!Adapter Error!! - %s", err)
		return nil
	}

	log.Println("connected to Rabbit Adapter")

	return adapter
}

func getSQSAdapter(config configuration.BrokerConfiguration) *adapters.SQSAdapter {
	adapter := adapters.NewSQSAdapter(config)
	if adapter == nil {
		log.Printf("!!Adapter Error!! - SQS")
	}
	return adapter
}

func setupRestEndpoints(e *echo.Echo, brokerAdapterManager service.BrokerAdapterManager) {
	// Get all brokers
	e.GET("brokers", brokerAdapterManager.GetAllBrokers)
	// Get all service for a particular queue associated with a broker
	e.GET(fmt.Sprintf("%s/:%s/%s/:%s/%s", "brokers", "brokerID", "queues", "queueName", "messages"), brokerAdapterManager.GetAllMessages)
	// Get all queues from a particular broker
	e.GET(fmt.Sprintf("%s/:%s/%s", "brokers", "brokerID", "queues"), brokerAdapterManager.GetAllQueues)
	// Remove all items from a queue from a particular broker
	e.DELETE(fmt.Sprintf("%s/:%s/%s/:%s", "brokers", "brokerID", "queues", "queueName"), brokerAdapterManager.PurgeFromQueue)
	// Remove a specific message from a queue from a particular broker
	e.DELETE(fmt.Sprintf("%s/:%s/%s/:%s/%s/:%s", "brokers", "brokerID", "queues", "queueName", "messages", "messageID"), brokerAdapterManager.DeleteMessageFromQueue)
	// Remove a list of items from a queue from a particular broker
	e.DELETE(fmt.Sprintf("%s/:%s/%s/:%s/%s", "brokers", "brokerID", "queues", "queueName", "messages"), brokerAdapterManager.DeleteMessagesFromQueue)
	//Move a specific message from a queue to another queue
	e.POST(fmt.Sprintf("%s/:%s/%s/:%s/%s/:%s/%s/:%s", "brokers", "brokerID", "queues", "queueName", "toqueue", "toQueueName", "messages", "messageID"), brokerAdapterManager.MoveMessage)
	//Move a list messages from a queue to another queue
	e.POST(fmt.Sprintf("%s/:%s/%s/:%s/%s/:%s/%s", "brokers", "brokerID", "queues", "queueName", "toqueue", "toQueueName", "messages"), brokerAdapterManager.MoveMessages)
}
