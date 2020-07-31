package adapters

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	amqp9 "github.com/streadway/amqp"
	"gitlab.com/ciorg/bridge/brokerUI/broker-service/pkg/structs"
)

type RabbitMQAdapter struct {
	username         string
	pwd              string
	consoleURL       string
	host             string
	url              string
	getAMQPPublisher func() (RabbitMQChannel, error)
}

// RabbitMQChannel is an interface with the methods on amqp.Channel that are necessary for sending notifications.
// (This is not the interface you are looking for...if implementing a new broker)
type RabbitMQChannel interface {
	io.Closer
	Confirm(noWait bool) error
	NotifyConfirm(ack, nack chan uint64) (chan uint64, chan uint64)
	Publish(exchange, key string, mandatory, immediate bool, msg amqp9.Publishing) error
	ExchangeDeclare(name, kind string, durable, autoDelete, internal, noWait bool, args amqp9.Table) error
	QueueDeclare(name string, durable, autoDelete, exclusive, noWait bool, args amqp9.Table) (amqp9.Queue, error)
	QueueBind(name, key, exchange string, noWait bool, args amqp9.Table) error
	Qos(prefetchCount, prefetchSize int, global bool) error
	Consume(queue, consumer string, autoAck, exclusive, noLocal, noWait bool, args amqp9.Table) (<-chan amqp9.Delivery, error)
	Cancel(consumer string, noWait bool) error
}

// RabbitMQGetMessagesRequestBody is used in a get messages request
var RabbitMQGetMessagesRequestBody = struct {
	Count    string `json:"count"`
	Ackmode  string `json:"ackmode"`
	Encoding string `json:"encoding"`
	Truncate int    `json:"truncate"`
}{
	"50000",            //limit number of messages to get at once
	"ack_requeue_true", //don't actually remove messages from queue
	"auto",
	50000,
}

// RabbitMQRemoveOneMessageRequestBody is used to remove a single message from a queue
var RabbitMQRemoveOneMessageRequestBody = struct {
	Count    string `json:"count"`
	Ackmode  string `json:"ackmode"`
	Encoding string `json:"encoding"`
	Truncate int    `json:"truncate"`
}{
	"1",                 //limit number of messages to get at once
	"ack_requeue_false", //remove message from queue
	"auto",
	50000,
}

// RabbitQueueInfo meta data for a RabbitMQ queue
type RabbitQueueInfo []struct {
	Consumers int    `json:"consumers"`
	Messages  int    `json:"messages"`
	Name      string `json:"name"`
	Vhost     string `json:"vhost"`
}

// RabbitMessages for parsing a collection of messages
type RabbitMessages []struct {
	QueueName  string `json:"routing_key"`
	Properties struct {
		Headers struct {
			CorrelationID string `json:"correlationID"`
			MessageID     string `json:"messageID"`
			Timestamp     string `json:"timestamp"`
		} `json:"headers"`
	} `json:"properties"`
	Body string `json:"payload"`
}

// RabbitPublishMessageRequestBody for publishing a message to a queue
type RabbitPublishMessageRequestBody struct {
	Properties struct {
		Headers struct {
			CorrelationID string `json:"correlationID"`
			MessageID     string `json:"messageID"`
			Timestamp     string `json:"timestamp"`
		} `json:"headers"`
	} `json:"properties"`
	RoutingKey      string `json:"routing_key"`
	Payload         string `json:"payload"`
	PayloadEncoding string `json:"payload_encoding"`
}

// Returns a RabbitMQ AMQP0.9 adapter:
// brokerURL: the AMQP URL for the broker (required if planning on using operations Move, MoveOne, DeleteOne, DeleteMany)
// consoleURL: the HTTP URL for the broker (required for all operations)
// username, pwd: username and password credentials required to connect
// host: virtual hostname for rabbitMQ
func NewRabbitMQAdapter(ctx context.Context, brokerURL, consoleURL, username, pwd, host string) (*RabbitMQAdapter, error) {

	url := brokerURL
	url = strings.Replace(url, "amqp://", "", 1)
	fullURL := username + ":" + pwd + "@" + url

	dialURL := fmt.Sprintf("amqp://%s", fullURL)
	amqpConn, err := amqp9.Dial(dialURL)
	if err != nil {
		log.Printf("Error connecting to AMQP 0.9 broker with %s: %s", dialURL, err)
	} else {
		log.Println("Connected to AMQP 0.9 broker with", dialURL)
	}

	return &RabbitMQAdapter{
		username:   username,
		pwd:        pwd,
		consoleURL: consoleURL,
		url:        brokerURL,
		host:       host,
		getAMQPPublisher: func() (RabbitMQChannel, error) {
			return amqpConn.Channel()
		}}, nil
}

func (r *RabbitMQAdapter) GetAllMessages(ctx context.Context, queueName string) ([]structs.StandardMessage, error) {

	httpClient := &http.Client{Timeout: time.Second * 10}
	var resp *http.Response

	url := fmt.Sprintf("%s/api/queues/%s/%s/get", r.consoleURL, r.host, queueName)
	log.Printf("attempting to get queue information from %s", url)
	body, err := json.Marshal(RabbitMQGetMessagesRequestBody)
	req, err := http.NewRequest("POST", url, strings.NewReader(string(body)))
	if err != nil {
		log.Printf("we were unable to get a http.NewRequest for consoleURL %s, error is %s", url, err.Error())
		return nil, nil
	}
	req.SetBasicAuth(r.username, r.pwd)

	resp, err = httpClient.Do(req)
	if err != nil {
		log.Printf("error returned from this attempt was %s", err.Error())
		return nil, nil
	}

	if resp == nil {
		log.Printf(fmt.Sprintf("we were not able to retrieve info about the queue from the console consoleURL: %s", url))
		return nil, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, nil
	}

	respbody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	messages := RabbitMessages{}

	err = json.Unmarshal(respbody, &messages)
	if err != nil {
		fmt.Printf("error: %v", err)
		return nil, err
	}

	queueInfoResult := []structs.StandardMessage{}

	for _, message := range messages {
		queueInfoResult = append(queueInfoResult, structs.StandardMessage{
			MessageID: message.Properties.Headers.MessageID,
			Headers:   map[string]string{"CorrelationID": message.Properties.Headers.CorrelationID},
			Body:      message.Body,
		})
	}

	return queueInfoResult, nil
}

func (r *RabbitMQAdapter) GetAllQueues(ctx context.Context) ([]Queue, error) {

	resp, err := r.getQueues()
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf(fmt.Sprintf("bad status code"))
		return nil, nil
	}

	respbody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Printf(fmt.Sprintf("read response failed"))
		return nil, err
	}

	rabbitQueueData := RabbitQueueInfo{}

	err = json.Unmarshal(respbody, &rabbitQueueData)
	if err != nil {
		fmt.Printf("error: %v", err)
		return nil, err
	}

	queueInfoResult := []Queue{}

	for _, queue := range rabbitQueueData {
		queueInfoResult = append(queueInfoResult, Queue{
			Name: queue.Name,
			Info: map[string]string{"Size": fmt.Sprintf("%d", queue.Messages)},
		})
	}

	return queueInfoResult, nil
}

func (r *RabbitMQAdapter) Move(ctx context.Context, fromQueue string, toQueue string, messageIDs []string) []error {
	errors := []error{}
	for _, messageID := range messageIDs {
		if err := r.MoveOne(ctx, fromQueue, toQueue, messageID); err != nil {
			errors = append(errors, err)
		}
	}
	return errors
}

func (r *RabbitMQAdapter) MoveOne(ctx context.Context, fromQueue string, toQueue string, messageID string) error {
	messagesToRequeue := []RabbitMessages{}
	queueLength := r.getQueueLength(ctx, fromQueue)
	log.Printf("queue length is: %d", queueLength)

	for i := 0; i < queueLength; i++ {
		// removing a message from fromQueue
		err, resp := r.removeOneMessageFromQueue(fromQueue)
		if err != nil {
			log.Printf("failed to remove message from %s", fromQueue)
			continue
		}
		if resp == nil {
			log.Printf("response was nil - failed to remove message from %s", fromQueue)
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			continue
		}

		respbody, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			continue
		}

		rabbitMessages := RabbitMessages{}

		err = json.Unmarshal(respbody, &rabbitMessages)
		if err != nil {
			fmt.Printf("error: %v", err)
			continue
		}

		if len(rabbitMessages) != 1 {
			log.Printf("Loop: Expected 1 message and got %d", len(rabbitMessages))
			continue
		}
		rabbitMessage := rabbitMessages[0]
		if messageID == rabbitMessage.Properties.Headers.MessageID {
			// move to new queue
			log.Printf("attempting to publish message %+v to queue: %s", rabbitMessage, toQueue)
			err := r.publishMessage(rabbitMessages, toQueue)
			if err != nil {
				messagesToRequeue = append(messagesToRequeue, rabbitMessages)
				log.Printf("Could not move to %s, requeued to %s: Error: %s", toQueue, fromQueue, err.Error())
			}
		} else {
			messagesToRequeue = append(messagesToRequeue, rabbitMessages)
			log.Printf("added message: %+v to the requeueMessages list", rabbitMessages)
		}
	}

	err := r.requeueMessages(messagesToRequeue, fromQueue)
	if err != nil {
		return err
	}

	return nil
}

func (r *RabbitMQAdapter) Purge(ctx context.Context, queueName string) error {
	httpClient := &http.Client{Timeout: time.Second * 10}
	var resp *http.Response

	url := fmt.Sprintf("%s/api/queues/%s/%s/contents", r.consoleURL, r.host, queueName)
	log.Printf("attempting to purge queue information from %s", url)
	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		log.Printf("we were unable to get a http.NewRequest for consoleURL %s, error is %s", url, err.Error())
		return err
	}
	req.SetBasicAuth(r.username, r.pwd)

	resp, err = httpClient.Do(req)
	if err != nil {
		log.Printf("error returned from this attempt was %s", err.Error())
		return err
	}

	if resp == nil {
		log.Printf(fmt.Sprintf("we were not able to purge the queue from the console consoleURL: %s", url))
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf(fmt.Sprintf("bad status code from the console consoleURL: %s", url))
		return nil
	}
	return nil
}

func (r *RabbitMQAdapter) DeleteOne(ctx context.Context, queueName string, messageID string) error {
	messagesToRequeue := []RabbitMessages{}
	queueLength := r.getQueueLength(ctx, queueName)
	log.Printf("queue %s length is: %d", queueName, queueLength)

	for i := 0; i < queueLength; i++ {
		// removing a message from fromQueue
		err, resp := r.removeOneMessageFromQueue(queueName)
		if err != nil {
			log.Printf("failed to remove message from %s", queueName)
			continue
		}
		if resp == nil {
			log.Printf("response was nil - failed to remove message from %s", queueName)
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			fmt.Printf("bad status code: %v", err)
			continue
		}

		respbody, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			fmt.Printf("resp body error: %v", err)
			continue
		}

		rabbitMessages := RabbitMessages{}

		err = json.Unmarshal(respbody, &rabbitMessages)
		if err != nil {
			fmt.Printf("unmarshal error: %v", err)
			continue
		}

		if len(rabbitMessages) != 1 {
			log.Printf("Loop: Expected 1 message and got %d", len(rabbitMessages))
			continue
		}
		rabbitMessage := rabbitMessages[0]
		if messageID != rabbitMessage.Properties.Headers.MessageID {
			log.Printf("added message: %+v to the requeueMessages list", rabbitMessages)
			messagesToRequeue = append(messagesToRequeue, rabbitMessages)
		}
	}

	err := r.requeueMessages(messagesToRequeue, queueName)
	if err != nil {
		fmt.Printf("requeue error: %v", err)
		return err
	}

	return nil
}

func (r *RabbitMQAdapter) DeleteMany(ctx context.Context, queueName string, messageIDs []string) []error {
	errors := []error{}
	for _, messageID := range messageIDs {
		if err := r.DeleteOne(ctx, queueName, messageID); err != nil {
			errors = append(errors, err)
		}
	}
	return errors
}

func (r *RabbitMQAdapter) getQueues() (*http.Response, error) {
	httpClient := &http.Client{Timeout: time.Second * 10}
	var resp *http.Response

	url := fmt.Sprintf("%s/api/queues/%s", r.consoleURL, r.host)
	log.Printf("attempting to get queue information from %s", url)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Printf("we were unable to get a http.NewRequest for consoleURL %s, error is %s", url, err.Error())
		return nil, err
	}
	req.SetBasicAuth(r.username, r.pwd)
	resp, err = httpClient.Do(req)
	if err != nil {
		log.Printf("error returned from this attempt was %s", err.Error())
		return nil, err
	}
	if resp == nil {
		log.Printf(fmt.Sprintf("we were not able to retrieve info about the queue from the console consoleURL: %s", url))
		return nil, errors.New(fmt.Sprintf("we were not able to retrieve info about the queue from the console consoleURL: %s", url))
	}
	return resp, nil
}

func (r *RabbitMQAdapter) getQueueLength(ctx context.Context, queueName string) int {
	queues, err := r.GetAllQueues(ctx)
	if err != nil {
		log.Printf("Could not retrieve queues from GetAllQueues")
		return 0
	}

	for _, q := range queues {
		if q.Name == queueName {
			i, ok := q.Info["Size"]
			if !ok {
				log.Printf("Error retrieving queue from queue list")
				continue
			}
			queueSize, err := strconv.Atoi(i)
			if err != nil {
				log.Printf("Error converting queue size string to int")
				return 0
			}
			log.Printf("found a queue: %s", queueName)
			return queueSize
		}
	}
	return 0
}

func (r *RabbitMQAdapter) requeueMessages(messages []RabbitMessages, queueName string) error {
	for _, message := range messages {
		err := r.publishMessage(message, queueName)
		if err != nil {
			log.Printf("Unable to requeue message to %s: %+v", queueName, message)
		}
	}

	return nil
}

func (r *RabbitMQAdapter) publishMessage(message RabbitMessages, toQueue string) error {
	return r.sendMessageAMQP(message, toQueue)
}

func (r *RabbitMQAdapter) sendMessageHTTP(message RabbitMessages, toQueue string) error {
	httpClient := &http.Client{Timeout: time.Second * 10}
	var resp *http.Response

	url := fmt.Sprintf("%s/api/exchanges/%s/%s/publish", r.consoleURL, r.host, toQueue)
	log.Printf("attempting to get queue information from %s", url)

	if len(message) != 1 {
		log.Printf("publish: Expected 1 message and got %d", len(message))
		return errors.New(fmt.Sprintf("Expected 1 message and got %d", len(message)))
	}
	rabbitMessage := message[0]
	body, err := json.Marshal(RabbitPublishMessageRequestBody{
		Properties:      rabbitMessage.Properties,
		RoutingKey:      toQueue,
		Payload:         rabbitMessage.Body,
		PayloadEncoding: "string",
	})
	req, err := http.NewRequest("POST", url, strings.NewReader(string(body)))
	if err != nil {
		log.Printf("we were unable to get a http.NewRequest for consoleURL %s, error is %s", url, err.Error())
		return err
	}
	req.SetBasicAuth(r.username, r.pwd)
	resp, err = httpClient.Do(req)
	if err != nil {
		log.Printf("error returned from this attempt was %s", err.Error())
		return err
	}
	if resp == nil {
		return errors.New(fmt.Sprintf("we were not able to retrieve info about the queue from the console consoleURL: %s", url))
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return errors.New("status not StatusOK")

	}
	return nil
}

func (r *RabbitMQAdapter) sendMessageAMQP(message RabbitMessages, toQueue string) error {

	//parse message for sending via AMQP09
	if len(message) != 1 {
		log.Printf("publish: Expected 1 message and got %d", len(message))
		return errors.New(fmt.Sprintf("Expected 1 message and got %d", len(message)))
	}
	rabbitMessage := message[0]

	timestamp, err := time.Parse("2006-01-02T15:04:05.000Z", rabbitMessage.Properties.Headers.Timestamp)
	if err != nil {
		timestamp = time.Now().UTC()
	}

	msg := amqp9.Publishing{
		MessageId:     rabbitMessage.Properties.Headers.MessageID,
		CorrelationId: rabbitMessage.Properties.Headers.CorrelationID,
		Body:          []byte(rabbitMessage.Body),
		DeliveryMode:  amqp9.Persistent,
		ContentType:   "application/json",
		Timestamp:     timestamp,
		Headers: amqp9.Table{
			"MessageID":     rabbitMessage.Properties.Headers.MessageID,
			"CorrelationID": rabbitMessage.Properties.Headers.CorrelationID,
			"Timestamp":     rabbitMessage.Properties.Headers.Timestamp,
		},
	}

	//connect channel
	publisher, err := r.getAMQPPublisher()
	if err != nil {
		return err
	}
	defer publisher.Close()

	// Set to Confirm mode, so we get delivery confirmation
	if err = publisher.Confirm(false); err != nil {
		return err
	}

	// Get channels for delivery confirmation/rejection
	ack, nack := publisher.NotifyConfirm(make(chan uint64, 1), make(chan uint64, 1))

	// Send each Message one at a time (easier to confirm delivery)
	if err = publisher.Publish(toQueue, toQueue, true, false, msg); err != nil {
		return err
	} else {

		// We don't want to wait on the Message delivery forever
		timeout := time.After(30 * time.Second)

		// In the case of a failure or timeout, log entire Message so it can be recovered
		select {
		case _ = <-ack:
			return nil
		case response := <-nack:
			err = errors.New(fmt.Sprintf("Message ID: %s; Nack: %+v", msg.MessageId, response))
			return err
		case _ = <-timeout:
			err = errors.New(fmt.Sprintf("Message ID: %s; Timeout", msg.MessageId))
			return err
		}
	}

}

func (r *RabbitMQAdapter) removeOneMessageFromQueue(fromQueue string) (error, *http.Response) {
	httpClient := &http.Client{Timeout: time.Second * 10}
	var resp *http.Response

	url := fmt.Sprintf("%s/api/queues/%s/%s/get", r.consoleURL, r.host, fromQueue)
	log.Printf("attempting to get queue information from %s", url)
	body, err := json.Marshal(RabbitMQRemoveOneMessageRequestBody)
	req, err := http.NewRequest("POST", url, strings.NewReader(string(body)))
	if err != nil {
		log.Printf("we were unable to get a http.NewRequest for consoleURL %s, error is %s", url, err.Error())
		return nil, nil
	}
	req.SetBasicAuth(r.username, r.pwd)
	resp, err = httpClient.Do(req)
	if err != nil {
		log.Printf("error returned from this attempt was %s", err.Error())
		return err, nil
	}
	if resp == nil {
		log.Printf(fmt.Sprintf("we were not able to retrieve info about the queue from the console consoleURL: %s", url))
	}
	return err, resp
}
