package adapters

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gitlab.com/ciorg/bridge/brokerUI/broker-service/pkg/structs"
)

type MockAdapter struct {
	MockMove           func(ctx context.Context, messageIDs []string, fromQueue string, toQueue string) []error
	MockMoveOne        func(ctx context.Context, messageID string, fromQueue string, toQueue string) error
	MockPurge          func(ctx context.Context, queueName string) error
	MockDeleteOne      func(ctx context.Context, messageID string, queueName string) error
	MockDeleteMany     func(ctx context.Context, messageIDs []string, queueName string) error
	MockGetAllMessages func(ctx context.Context, queueName string) ([]structs.StandardMessage, error)
	MockGetAllQueues   func(ctx context.Context) ([]string, error)
}

func (m *MockAdapter) Move(ctx context.Context, fromQueue string, toQueue string, messageIDs []string) []error {
	return nil
}

func (m *MockAdapter) MoveOne(ctx context.Context, fromQueue string, toQueue string, messageID string) error {
	return nil
}

func (m *MockAdapter) Purge(ctx context.Context, queueName string) error {
	return nil
}

func (m *MockAdapter) DeleteOne(ctx context.Context, queueName string, messageID string) error {
	return nil
}

func (m *MockAdapter) DeleteMany(ctx context.Context, queueName string, messageIDs []string) []error {
	return nil
}

func (m *MockAdapter) GetAllMessages(ctx context.Context, queueName string) ([]structs.StandardMessage, error) {
	message := structs.StandardMessage{
		MessageID: uuid.New().String(),
		Timestamp: time.Now().UTC(),
		Headers: map[string]string{
			"blah":     "blah",
			"blahblah": "blahblah",
		},
		Body: "Hello this is a very important message. Please don't ignore it",
	}

	message2 := structs.StandardMessage{
		MessageID: uuid.New().String(),
		Timestamp: time.Now().UTC().Add(-48 * time.Hour),
		Headers: map[string]string{
			"blah1":     "blah1",
			"blahblah1": "blahblah1",
		},
		Body: "Hello this is another very important message. Please don't ignore it",
	}

	messages := []structs.StandardMessage{message, message2}
	return messages, nil
}

func (m *MockAdapter) GetAllQueues(ctx context.Context) ([]Queue, error) {
	queues := []Queue{
		{Name: "test_queue_1", Info: map[string]string{"Size": "2"}},
		{Name: "test_queue_2", Info: map[string]string{"Size": "2"}},
	}
	return queues, nil
}
