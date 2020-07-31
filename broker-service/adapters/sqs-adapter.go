package adapters

import (
	"context"
	"errors"
	"log"
	"net/url"
	"os"
	"time"

	"gitlab.com/ciorg/bridge/brokerUI/broker-service/configuration"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sqs"
	"gitlab.com/ciorg/bridge/brokerUI/broker-service/pkg/structs"
)

func NewSQSAdapter(config configuration.BrokerConfiguration) *SQSAdapter {

	region := config.All["REGION"]
	accessKey := config.All["ACCESS_KEY"]
	secretKey := config.All["SECRET_KEY"]

	sessionConfig := &aws.Config{
		Region:      aws.String(region),
		Credentials: credentials.NewStaticCredentials(accessKey, secretKey, ""),
	}

	awsSession, err := session.NewSession(sessionConfig)
	if err != nil {
		log.Printf("error establishing AWS session: %s\n", err)
		return nil
	}

	return &SQSAdapter{awsSession: awsSession}

}

type SQSAdapter struct {
	awsSession *session.Session
}

func (s *SQSAdapter) GetAllMessages(ctx context.Context, encodedQueueName string) ([]structs.StandardMessage, error) {

	queueName, _ := url.QueryUnescape(encodedQueueName)

	messages := make([]structs.StandardMessage, 0)

	svc := sqs.New(s.awsSession, nil)
	receiveMessagesInput := &sqs.ReceiveMessageInput{
		AttributeNames: []*string{
			aws.String(sqs.MessageSystemAttributeNameSentTimestamp),
		},
		MessageAttributeNames: []*string{
			aws.String(sqs.QueueAttributeNameAll),
		},
		QueueUrl:            aws.String(queueName),
		MaxNumberOfMessages: aws.Int64(10), // max 10
		WaitTimeSeconds:     aws.Int64(3),  // max 20
		VisibilityTimeout:   aws.Int64(20), // max 20
	}

	receiveMessageOutput, err :=
		svc.ReceiveMessage(receiveMessagesInput)

	if err != nil {
		return nil, err
	}

	if receiveMessageOutput == nil || len(receiveMessageOutput.Messages) == 0 {
		return messages, nil
	}

	for _, message := range receiveMessageOutput.Messages {
		timestamp, _ := time.Parse("", *message.Attributes["SentTimestamp"])

		headers := make(map[string]string)
		for attributekey, attributeval := range message.Attributes {
			headers[attributekey] = *attributeval
		}
		for attributeKey, attributeVal := range message.MessageAttributes {
			headers[attributeKey] = attributeVal.String()
		}

		stdMsg := structs.StandardMessage{
			MessageID: *message.MessageId,
			Timestamp: timestamp,
			Headers:   headers,
			Body:      *message.Body,
		}

		messages = append(messages, stdMsg)
	}

	// NOTE: no need to NACK.  Message will automatically go back into queue in 20 seconds.

	// TODO this only gets 10 messages at a time.  Loop back and get more.  Might have to be in multiple go routines?

	return messages, nil
}

func (s *SQSAdapter) GetAllQueues(ctx context.Context) ([]Queue, error) {

	queues := make([]Queue, 0)
	svc := sqs.New(s.awsSession, nil)
	if svc == nil {
		log.Println("There was a problem getting a new SQS instance")
		os.Exit(1)
	}

	listQueuesOutput, err := svc.ListQueues(nil)
	if err != nil {
		return nil, err
	}
	if listQueuesOutput == nil || len(listQueuesOutput.QueueUrls) == 0 {
		return queues, nil
	}

	for _, queueUrl := range listQueuesOutput.QueueUrls {
		if queueUrl == nil {
			continue
		}
		queue := Queue{
			Name: *queueUrl,
			Info: nil,
		}
		queues = append(queues, queue)
	}

	return queues, nil
}

func (s *SQSAdapter) Move(ctx context.Context, fromEncodedQueueName string, toEncodedQueueName string, messageIDs []string) []error {
	result := make([]error, 1)
	result[1] = errors.New("Not implemented")
	return result
}

func (s *SQSAdapter) MoveOne(ctx context.Context, fromQueue string, toQueue string, messageID string) error {
	return errors.New("Not implemented")
}

func (s *SQSAdapter) Purge(ctx context.Context, encodedQueueName string) error {
	return errors.New("Not implemented")
}

func (s *SQSAdapter) DeleteOne(ctx context.Context, encodedQueueName string, messageID string) error {
	return errors.New("Not implemented")
}

func (s *SQSAdapter) DeleteMany(ctx context.Context, encodedQueueName string, messageIDs []string) []error {
	result := make([]error, 1)
	result[1] = errors.New("Not implemented")
	return result
}
