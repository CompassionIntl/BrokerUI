package adapters

import (
	"context"
	"fmt"
	"testing"
	"time"

	"gitlab.com/ciorg/bridge/brokerUI/broker-service/pkg/structs"

	"github.com/google/uuid"

	"github.com/Azure/go-amqp"
)

func TestActiveMQAdapter_GetAllMessages(t *testing.T) {

	t.Run("Boy Howdy", func(t *testing.T) {
		a, err := NewActiveMQAdapter(context.Background(), "amqp://localhost:5671", "admin", "admin", "http://localhost:8162", "admin", "admin", false)
		if err != nil {
			t.Fatalf("No connection. %s", err)
		}

		got, err := a.GetAllMessages(context.Background(), "Consumer.notifier.VirtualTopic.commitment_update_deadletter")
		if err != nil {
			t.Fatalf("Doom. %s", err)
		}
		fmt.Printf("%+v\n", got)
	})

}

func TestMoveMessages(t *testing.T) {

	queueName := "movetohere"
	deadLetterSuffix := "_fromhere"
	deadLetterQueue := queueName + deadLetterSuffix

	t.Run("I like to move it move it", func(tt *testing.T) {
		// testHelper := Amqp10TestHelper{}
		// _ = testHelper.DeleteQueue(queueName)
		// _ = testHelper.DeleteQueue(deadLetterQueue)
		// defer testHelper.DeleteQueue(queueName)
		// defer testHelper.DeleteQueue(deadLetterQueue)

		a, err := NewActiveMQAdapter(context.Background(), "amqp://localhost:5671", "admin", "admin", "http://localhost:8162", "admin", "admin", false)
		if err != nil {
			t.Fatalf("No connection. %s", err)
		}

		mySender, err, closeSession := a.getNewSender(context.Background(), deadLetterQueue)
		if err != nil {
			t.Fatalf("Doom. %s", err)
		}
		defer closeSession()

		msgCount := 5
		myMessages := make([]string, 0)
		for i := 0; i < msgCount; i++ {
			msg := amqp.Message{

				Data: [][]byte{[]byte(fmt.Sprintf("string %d", i))},
			}
			msg.Properties = &amqp.MessageProperties{
				MessageID:    uuid.New().String(),
				CreationTime: time.Now().UTC(),
			}
			msg.Header = &amqp.MessageHeader{
				Durable:       true,
				Priority:      0,
				TTL:           0,
				FirstAcquirer: false,
				DeliveryCount: 0,
			}

			_ = mySender.Send(context.Background(), &msg)
			myMessages = append(myMessages, fmt.Sprintf("%v", msg.Properties.MessageID))
		}

		errs := a.Move(context.Background(), deadLetterQueue, queueName, myMessages)
		if errs != nil {
			t.Error(errs[0])
		}

		messages, _ := a.GetAllMessages(context.Background(), queueName)

		messageMap := make(map[string]structs.StandardMessage)
		for _, message := range messages {
			messageMap[message.MessageID] = message
		}

		for _, msgId := range myMessages {
			_, ok := messageMap[msgId]
			if !ok {
				t.Errorf("Message %s not found in destination", msgId)
			}
		}
	})

}

func TestDeleteMessages(t *testing.T) {

	queueName := "todeletefrom"

	t.Run("I like to delete them", func(tt *testing.T) {
		// testHelper := Amqp10TestHelper{}
		// _ = testHelper.DeleteQueue(queueName)
		// _ = testHelper.DeleteQueue(deadLetterQueue)
		// defer testHelper.DeleteQueue(queueName)
		// defer testHelper.DeleteQueue(deadLetterQueue)

		a, err := NewActiveMQAdapter(context.Background(), "amqp://localhost:5671", "admin", "admin", "http://localhost:8162", "admin", "admin", false)
		if err != nil {
			t.Fatalf("No connection. %s", err)
		}

		mySender, err, closeSession := a.getNewSender(context.Background(), queueName)
		if err != nil {
			t.Fatalf("Doom. %s", err)
		}
		defer closeSession()

		msgCount := 5
		myMessages := make([]string, 0)
		for i := 0; i < msgCount; i++ {
			msg := amqp.Message{

				Data: [][]byte{[]byte(fmt.Sprintf("string %d", i))},
			}
			msg.Properties = &amqp.MessageProperties{
				MessageID:    uuid.New().String(),
				CreationTime: time.Now().UTC(),
			}
			msg.Header = &amqp.MessageHeader{
				Durable:       true,
				Priority:      0,
				TTL:           0,
				FirstAcquirer: false,
				DeliveryCount: 0,
			}

			_ = mySender.Send(context.Background(), &msg)
			myMessages = append(myMessages, fmt.Sprintf("%v", msg.Properties.MessageID))
		}

		errs := a.DeleteMany(context.Background(), queueName, myMessages)
		if errs != nil {
			t.Error(errs[0])
		}
	})

}

func TestPurgeMessages(t *testing.T) {

	queueName := "topurgefrom"

	t.Run("I like to purge them", func(tt *testing.T) {
		// testHelper := Amqp10TestHelper{}
		// _ = testHelper.DeleteQueue(queueName)
		// _ = testHelper.DeleteQueue(deadLetterQueue)
		// defer testHelper.DeleteQueue(queueName)
		// defer testHelper.DeleteQueue(deadLetterQueue)

		a, err := NewActiveMQAdapter(context.Background(), "amqp://localhost:5671", "admin", "admin", "http://localhost:8162", "admin", "admin", false)
		if err != nil {
			t.Fatalf("No connection. %s", err)
		}

		mySender, err, closeSession := a.getNewSender(context.Background(), queueName)
		if err != nil {
			t.Fatalf("Doom. %s", err)
		}
		defer closeSession()

		msgCount := 5

		for i := 0; i < msgCount; i++ {
			msg := amqp.Message{

				Data: [][]byte{[]byte(fmt.Sprintf("string %d", i))},
			}
			msg.Properties = &amqp.MessageProperties{
				MessageID:    uuid.New().String(),
				CreationTime: time.Now().UTC(),
			}
			msg.Header = &amqp.MessageHeader{
				Durable:       true,
				Priority:      0,
				TTL:           0,
				FirstAcquirer: false,
				DeliveryCount: 0,
			}

			_ = mySender.Send(context.Background(), &msg)

		}

		err = a.Purge(context.Background(), queueName)
		if err != nil {
			t.Error(err)
		}
	})

}
