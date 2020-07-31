package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"gitlab.com/ciorg/bridge/brokerUI/broker-service/pkg/structs"

	"github.com/labstack/echo"
	"gitlab.com/ciorg/bridge/brokerUI/broker-service/adapters"
)

type BrokerAdapterManager struct {
	MapBrokerNameToAdapter map[string]adapters.Adapter
}

func (b *BrokerAdapterManager) GetAllMessages(echoContext echo.Context) error {
	queueName := echoContext.Param("queueName")
	brokerID := echoContext.Param("brokerID")

	if queueName == "" {
		return echoContext.JSONPretty(http.StatusBadRequest, "no queue name given", "   ")
	}

	if brokerID == "" {
		return echoContext.JSONPretty(http.StatusBadRequest, "no broker name given", "   ")
	}

	brokerAdapter, ok := b.MapBrokerNameToAdapter[brokerID]
	if !ok {
		return echoContext.JSONPretty(http.StatusBadRequest, fmt.Sprintf("No connection found for %s", brokerID), "   ")
	}

	messages, err := brokerAdapter.GetAllMessages(context.Background(), queueName)
	if err != nil {
		return echoContext.JSONPretty(http.StatusInternalServerError, err.Error(), "   ")
	}

	err = echoContext.JSONPretty(http.StatusOK, messages, "   ")
	return err
}

func (b *BrokerAdapterManager) GetAllBrokers(echoContext echo.Context) error {
	var brokerAdapters []adapters.Broker

	for k, _ := range b.MapBrokerNameToAdapter {
		brokerAdapters = append(brokerAdapters, adapters.Broker{Name: k})
	}

	if len(brokerAdapters) == 0 {
		return echoContext.JSONPretty(http.StatusInternalServerError, nil, "  ")
	}

	err := echoContext.JSONPretty(http.StatusOK, brokerAdapters, "   ")
	return err
}

func (b *BrokerAdapterManager) GetAllQueues(echoContext echo.Context) error {
	brokerID := echoContext.Param("brokerID")

	if brokerID == "" {
		return echoContext.JSONPretty(http.StatusBadRequest, nil, "   ")
	}

	brokerAdapter, ok := b.MapBrokerNameToAdapter[brokerID]
	if !ok {
		return echoContext.JSONPretty(http.StatusBadRequest, nil, "   ")
	}

	queues, err := brokerAdapter.GetAllQueues(context.Background())
	if err != nil {
		return echoContext.JSONPretty(http.StatusInternalServerError, err.Error(), "   ")
	}

	err = echoContext.JSONPretty(http.StatusOK, queues, "   ")
	return err
}

func (b *BrokerAdapterManager) PurgeFromQueue(echoContext echo.Context) error {
	queueName := echoContext.Param("queueName")
	brokerID := echoContext.Param("brokerID")

	if queueName == "" {
		return echoContext.JSONPretty(http.StatusBadRequest, nil, "   ")
	}

	if brokerID == "" {
		return echoContext.JSONPretty(http.StatusBadRequest, nil, "   ")
	}

	brokerAdapter, ok := b.MapBrokerNameToAdapter[brokerID]
	if !ok {
		return echoContext.JSONPretty(http.StatusBadRequest, fmt.Sprintf("No connection found for %s", brokerID), "   ")
	}

	err := brokerAdapter.Purge(context.Background(), queueName)
	if err != nil {
		return echoContext.JSONPretty(http.StatusInternalServerError, err.Error(), "   ")
	}

	err = echoContext.JSONPretty(http.StatusOK, nil, "   ")
	return err
}

func (b *BrokerAdapterManager) DeleteMessageFromQueue(echoContext echo.Context) error {

	queueName := echoContext.Param("queueName")
	brokerID := echoContext.Param("brokerID")
	messageID := echoContext.Param("messageID")

	if queueName == "" {
		return echoContext.JSONPretty(http.StatusBadRequest, nil, "   ")
	}

	if brokerID == "" {
		return echoContext.JSONPretty(http.StatusBadRequest, nil, "   ")
	}

	if messageID == "" {
		return echoContext.JSONPretty(http.StatusBadRequest, nil, "   ")
	}

	brokerAdapter, ok := b.MapBrokerNameToAdapter[brokerID]
	if !ok {
		return echoContext.JSONPretty(http.StatusBadRequest, fmt.Sprintf("No connection found for %s", brokerID), "   ")
	}

	err := brokerAdapter.DeleteOne(context.Background(), queueName, messageID)
	if err != nil {
		return echoContext.JSONPretty(http.StatusInternalServerError, err.Error(), "   ")
	}

	err = echoContext.JSONPretty(http.StatusOK, nil, "   ")
	return err
}

func (b *BrokerAdapterManager) DeleteMessagesFromQueue(echoContext echo.Context) error {

	queueName := echoContext.Param("queueName")
	brokerID := echoContext.Param("brokerID")
	body, err := getBody(echoContext)
	if err != nil {
		return echoContext.JSONPretty(http.StatusInternalServerError, err.Error(), "   ")
	}

	var req structs.RequestMessageIDs
	err = json.Unmarshal(body, &req)
	if err != nil {
		return echoContext.JSONPretty(http.StatusInternalServerError, err.Error(), "   ")
	}

	if queueName == "" {
		return echoContext.JSONPretty(http.StatusBadRequest, nil, "   ")
	}

	if brokerID == "" {
		return echoContext.JSONPretty(http.StatusBadRequest, nil, "   ")
	}

	brokerAdapter, ok := b.MapBrokerNameToAdapter[brokerID]
	if !ok {
		return echoContext.JSONPretty(http.StatusBadRequest, fmt.Sprintf("No connection found for %s", brokerID), "   ")
	}

	errs := brokerAdapter.DeleteMany(context.Background(), queueName, req.MessageIDs)
	if errs != nil {
		return echoContext.JSONPretty(http.StatusInternalServerError, createErrorStrings(errs), "   ")
	}

	err = echoContext.JSONPretty(http.StatusOK, nil, "   ")
	return err
}

func (b *BrokerAdapterManager) MoveMessage(echoContext echo.Context) error {

	queueName := echoContext.Param("queueName")
	toQueueName := echoContext.Param("toQueueName")
	brokerID := echoContext.Param("brokerID")
	messageID := echoContext.Param("messageID")

	if queueName == "" {
		return echoContext.JSONPretty(http.StatusBadRequest, nil, "   ")
	}

	if toQueueName == "" {
		return echoContext.JSONPretty(http.StatusBadRequest, nil, "   ")
	}

	if brokerID == "" {
		return echoContext.JSONPretty(http.StatusBadRequest, nil, "   ")
	}

	if messageID == "" {
		return echoContext.JSONPretty(http.StatusBadRequest, nil, "   ")
	}

	brokerAdapter, ok := b.MapBrokerNameToAdapter[brokerID]
	if !ok {
		return echoContext.JSONPretty(http.StatusBadRequest, fmt.Sprintf("No connection found for %s", brokerID), "   ")
	}

	err := brokerAdapter.MoveOne(context.Background(), queueName, toQueueName, messageID)
	if err != nil {
		return echoContext.JSONPretty(http.StatusInternalServerError, err.Error(), "   ")
	}

	err = echoContext.JSONPretty(http.StatusOK, nil, "   ")
	return err
}

func (b *BrokerAdapterManager) MoveMessages(echoContext echo.Context) error {

	queueName := echoContext.Param("queueName")
	toQueueName := echoContext.Param("toQueueName")
	brokerID := echoContext.Param("brokerID")
	body, err := getBody(echoContext)
	if err != nil {
		return echoContext.JSONPretty(http.StatusInternalServerError, err.Error(), "   ")
	}

	var req structs.RequestMessageIDs
	err = json.Unmarshal(body, &req)
	if err != nil {
		return echoContext.JSONPretty(http.StatusInternalServerError, err.Error(), "   ")
	}

	if queueName == "" {
		return echoContext.JSONPretty(http.StatusBadRequest, nil, "   ")
	}

	if toQueueName == "" {
		return echoContext.JSONPretty(http.StatusBadRequest, nil, "   ")
	}

	if brokerID == "" {
		return echoContext.JSONPretty(http.StatusBadRequest, nil, "   ")
	}

	brokerAdapter, ok := b.MapBrokerNameToAdapter[brokerID]
	if !ok {
		return echoContext.JSONPretty(http.StatusBadRequest, fmt.Sprintf("No connection found for %s", brokerID), "   ")
	}

	errs := brokerAdapter.Move(context.Background(), queueName, toQueueName, req.MessageIDs)
	if errs != nil {
		stringErrs := createErrorStrings(errs)
		return echoContext.JSONPretty(http.StatusInternalServerError, stringErrs, "   ")
	}

	err = echoContext.JSONPretty(http.StatusOK, nil, "   ")
	return err
}

func createErrorStrings(errs []error) []string {
	var stringErrors []string
	for _, err := range errs {
		stringErrors = append(stringErrors, err.Error())
	}

	return stringErrors
}

func getBody(echoContext echo.Context) ([]byte, error) {
	reqBody := echoContext.Request().Body
	body, err := ioutil.ReadAll(reqBody)
	if err != nil {
		return nil, err
	}
	return body, nil
}
