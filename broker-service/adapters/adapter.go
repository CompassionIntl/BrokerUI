package adapters

import (
	"context"

	"gitlab.com/ciorg/bridge/brokerUI/broker-service/pkg/structs"
)

type Adapter interface {
	GetAllMessages(ctx context.Context, queueName string) ([]structs.StandardMessage, error)
	GetAllQueues(ctx context.Context) ([]Queue, error)
	Move(ctx context.Context, fromQueue string, toQueue string, messageIDs []string) []error
	MoveOne(ctx context.Context, fromQueue string, toQueue string, messageID string) error
	Purge(ctx context.Context, queueName string) error
	DeleteOne(ctx context.Context, queueName string, messageID string) error
	DeleteMany(ctx context.Context, queueName string, messageIDs []string) []error
}

type Queue struct {
	Name string
	Info map[string]string
}

type Broker struct {
	Name string
	Info map[string]string
}
