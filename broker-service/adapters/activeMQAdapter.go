package adapters

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/Azure/go-amqp"
	"gitlab.com/ciorg/bridge/brokerUI/broker-service/pkg/structs"
)

type QueuesXMLData struct {
	XMLName xml.Name `xml:"queues"`
	Text    string   `xml:",chardata"`
	Queue   []struct {
		Text  string `xml:",chardata"`
		Name  string `xml:"name,attr"`
		Stats struct {
			Text          string `xml:",chardata"`
			Size          string `xml:"size,attr"`
			ConsumerCount string `xml:"consumerCount,attr"`
			EnqueueCount  string `xml:"enqueueCount,attr"`
			DequeueCount  string `xml:"dequeueCount,attr"`
		} `xml:"stats"`
		Feed struct {
			Text string `xml:",chardata"`
			Atom string `xml:"atom"`
			Rss  string `xml:"rss"`
		} `xml:"feed"`
	} `xml:"queue"`
}

func NewActiveMQAdapter(ctx context.Context, brokerUrl, userName, passwd,
	brokerConsoleUrl, consoleUser, consolePasswd string, useTLS bool) (*ActiveMQAdapter, error) {

	var client *amqp.Client
	var err error

	brokerUrls := strings.Split(brokerUrl, ",")
	brokerConsoleUrls := strings.Split(brokerConsoleUrl, ",")

	for _, brokerUrl := range brokerUrls {
		if useTLS {
			log.Println("Attempting to connect to broker using TLS", brokerUrl)
			client, err = amqp.Dial(brokerUrl, amqp.ConnSASLPlain(userName, passwd), amqp.ConnTLS(true), amqp.ConnIdleTimeout(0))
		} else {
			log.Println("Attempting to connect to broker using plain", brokerUrl)
			client, err = amqp.Dial(brokerUrl, amqp.ConnSASLPlain(userName, passwd), amqp.ConnIdleTimeout(0))
		}
		if err == nil {
			break
		}
	}

	if err != nil {
		return nil, err
	}

	return &ActiveMQAdapter{
		client:           client,
		brokerConsoleUrls: brokerConsoleUrls,
		brokerConsoleUsr: consoleUser,
		brokerConsolePwd: consolePasswd,
	}, nil
}

type ActiveMQAdapter struct {
	client           			*amqp.Client
	brokerConsoleUrls 			[]string
	brokerConsoleUsr 			string
	brokerConsolePwd 			string
}

func (a *ActiveMQAdapter) GetAllMessages(ctx context.Context, queueName string) ([]structs.StandardMessage, error) {
	session, err, closeSession := a.getSession(ctx)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("Get new session failed: %s", err.Error()))
	}
	defer closeSession()

	var receiver *amqp.Receiver
	receiver, err = session.NewReceiver(
		amqp.LinkSourceAddress(queueName),
		amqp.LinkCredit(10),
	)
	if err != nil {
		log.Printf("unable to get new receiver, error is %s", err.Error())
		return nil, errors.New(fmt.Sprintf("getNewReceiver failed: %s", err.Error()))
	}

	defer func() {
		closeContext, closeCancel := context.WithTimeout(ctx, 10*time.Second)
		err := receiver.Close(closeContext)
		closeCancel()
		if err != nil {
			fmt.Printf("Unable to close the receiver: %s", err)
		}
	}()

	fmt.Println("receiver succeeded")

	var messages []*amqp.Message
	numOfErrors := 0
	for {
		ctxForReceive, cancelFunction := context.WithTimeout(ctx, 1*time.Second)
		msg, err := receiver.Receive(ctxForReceive)
		if ctxForReceive.Err() != nil {
			break
		}
		cancelFunction()
		if err != nil {
			fmt.Printf("Unable to receive messages: %s", err)
			numOfErrors++
			if numOfErrors > 10 { // TODO make configurable?
				return nil, errors.New("unable to receive messages")
			}
			continue
		}

		messages = append(messages, msg)

	}

	// release the message (don't remove it from the queue)
	for _, msg := range messages {
		msg.Release()
	}

	stdMsgs, _ := a.convertMessagesToStandardMessage(ctx, messages)

	return stdMsgs, nil
}

func (a *ActiveMQAdapter) Move(ctx context.Context, fromQueue string, toQueue string, messageIDs []string) []error {
	// var moveErrors []error
	// for i, _ := range messageIDs {
	// 	if err := a.MoveOne(ctx, fromQueue, toQueue, messageIDs[i]); err != nil {
	// 		moveErrors = append(moveErrors, err)
	// 	}
	// }
	// return moveErrors

	var moveErrors []error

	session, err, closeSession := a.getSession(ctx)
	if err != nil {
		return append(moveErrors, errors.New(fmt.Sprintf("Get new session failed: %s", err.Error())))
	}
	defer closeSession()
	var receiver *amqp.Receiver
	receiver, err = session.NewReceiver(
		amqp.LinkSourceAddress(fromQueue),
		amqp.LinkCredit(10),
	)
	if err != nil {
		return append(moveErrors, errors.New(fmt.Sprintf("geting new receiver failed: %s", err.Error())))
	}

	defer func() {
		closeContext, closeCancel := context.WithTimeout(ctx, 10*time.Second)
		err := receiver.Close(closeContext)
		closeCancel()
		if err != nil {
			fmt.Printf("Unable to close the receiver: %s", err)
		}
	}()

	sender, err, closeSession := a.getNewSender(ctx, toQueue)
	if err != nil {
		return []error{errors.New(fmt.Sprintf("Error initiating sender: %s", err))}
	}
	defer closeSession()
	defer sender.Close(ctx)

	rssData, err := a.retrieveRssDataForQueue(fromQueue)
	if err != nil {
		return append(moveErrors, err)
	}
	numberOfMessages := len(rssData.Channel.Item)
	if numberOfMessages == 0 {
		return append(moveErrors, errors.New("no items found in queue"))
	}

	messages := make(map[string]*amqp.Message)
	for i := 0; i < numberOfMessages; i++ {
		ctxForReceive, cancelFunction := context.WithTimeout(ctx, 1*time.Second)
		msg, err := receiver.Receive(ctxForReceive)
		if ctxForReceive.Err() != nil {
			log.Println("break from context")
			cancelFunction()
			moveErrors = append(moveErrors, ctxForReceive.Err())
			break
		}
		cancelFunction()
		if err != nil {
			fmt.Printf("Unable to receive messages: %s", err)
			return append(moveErrors, ctxForReceive.Err())
		}

		var msgId string
		var ok bool
		if msgId, ok = msg.Properties.MessageID.(string); !ok {
			continue
		}

		messages[msgId] = msg
	}

	for _, msgId := range messageIDs {
		var msg *amqp.Message
		var ok bool
		if msg, ok = messages[msgId]; !ok {
			moveErrors = append(moveErrors, fmt.Errorf("Did not find message %s", msgId))
			continue
		}
		if err := sender.Send(ctx, msg); err != nil {
			log.Printf("error trying to send message %s, error is %s", msgId, err)
			moveErrors = append(moveErrors, err)
			continue
		}
		if err := msg.Accept(); err != nil {
			log.Printf("error trying to accept message %s, error is %s", msgId, err)
			continue
		}

		delete(messages, msgId)
	}

	for _, msg := range messages {
		// release the message if it wasn't the one we were looking for
		if err := msg.Release(); err != nil {
			log.Printf("error trying to release message %s", err)
		}
	}

	return moveErrors
}

func (a *ActiveMQAdapter) MoveOne(ctx context.Context, fromQueue string, toQueue string, messageID string) error {
	session, err, closeSession := a.getSession(ctx)
	if err != nil {
		return errors.New(fmt.Sprintf("Get new session failed: %s", err.Error()))
	}
	defer closeSession()
	var receiver *amqp.Receiver
	receiver, err = session.NewReceiver(
		amqp.LinkSourceAddress(fromQueue),
		amqp.LinkCredit(10),
	)
	if err != nil {
		log.Printf("unable to get new receiver, error is %s", err.Error())
		return errors.New(fmt.Sprintf("geting new receiver failed: %s", err.Error()))
	}

	defer func() {
		closeContext, closeCancel := context.WithTimeout(ctx, 10*time.Second)
		err := receiver.Close(closeContext)
		closeCancel()
		if err != nil {
			fmt.Printf("Unable to close the receiver: %s", err)
		}
	}()
	defer receiver.Close(ctx)

	sender, err, closeSession := a.getNewSender(ctx, toQueue)
	if err != nil {
		return errors.New(fmt.Sprintf("Send errored: %s", err))
	}
	defer closeSession()
	defer sender.Close(ctx)

	rssData, err := a.retrieveRssDataForQueue(fromQueue)
	if err != nil {
		return err
	}
	numberOfMessages := len(rssData.Channel.Item)
	if numberOfMessages == 0 {
		return errors.New("no items found in queue")
	}

	for i := 0; i < numberOfMessages; i++ {
		ctxForReceive, cancelFunction := context.WithTimeout(ctx, 1*time.Second)
		msg, err := receiver.Receive(ctxForReceive)
		if ctxForReceive.Err() != nil {
			break
		}
		cancelFunction()
		if err != nil {
			fmt.Printf("Unable to receive messages: %s", err)
			continue
		}

		if fmt.Sprintf("%v", msg.Properties.MessageID) == messageID {
			if err := sender.Send(ctx, msg); err != nil {
				log.Printf("error trying to send message %s, error is %s", messageID, err)
				msg.Release()
				return err
			}
			if err := msg.Accept(); err != nil {
				log.Printf("error trying to accept message %s, error is %s", messageID, err)
				return err
			}
			// got the message we wanted and moved it, can stop processing now
			return nil
		}
		if err := msg.Release(); err != nil {
			log.Printf("error releasing message %s", err)
			return errors.New(fmt.Sprintf("unable to release message id %v", msg.Properties.MessageID))
		}

	}

	return nil
}

func (a *ActiveMQAdapter) Purge(ctx context.Context, queueName string) error {
	session, err, closeSession := a.getSession(ctx)
	if err != nil {
		return errors.New(fmt.Sprintf("Get new session failed: %s", err.Error()))
	}
	defer closeSession()
	var receiver *amqp.Receiver
	receiver, err = session.NewReceiver(
		amqp.LinkSourceAddress(queueName),
		amqp.LinkCredit(10),
	)
	if err != nil {
		log.Printf("unable to get new receiver, error is %s", err.Error())
		return errors.New(fmt.Sprintf("geting new receiver failed: %s", err.Error()))
	}

	defer func() {
		closeContext, closeCancel := context.WithTimeout(ctx, 10*time.Second)
		err := receiver.Close(closeContext)
		closeCancel()
		if err != nil {
			fmt.Printf("Unable to close the receiver: %s", err)
		}
	}()
	defer receiver.Close(ctx)

	rssData, err := a.retrieveRssDataForQueue(queueName)
	if err != nil {
		return err
	}
	numberOfMessages := len(rssData.Channel.Item)

	for i := 0; i < numberOfMessages; i++ {
		ctxForReceive, cancelFunction := context.WithTimeout(ctx, 1*time.Second)
		msg, err := receiver.Receive(ctxForReceive)
		if ctxForReceive.Err() != nil {
			break
		}
		cancelFunction()
		if err != nil {
			fmt.Printf("Unable to receive messages: %s", err)
			continue
		}

		if err := msg.Accept(); err != nil {
			log.Printf("error trying to accept message %v, error is %s", msg.Properties.MessageID, err)
			return err
		}

	}

	return nil
}

func (a *ActiveMQAdapter) DeleteOne(ctx context.Context, queueName string, messageID string) error {

	session, err, closeSession := a.getSession(ctx)
	if err != nil {
		return errors.New(fmt.Sprintf("Get new session failed: %s", err.Error()))
	}
	defer closeSession()
	var receiver *amqp.Receiver
	receiver, err = session.NewReceiver(
		amqp.LinkSourceAddress(queueName),
		amqp.LinkCredit(10),
	)
	if err != nil {
		return errors.New(fmt.Sprintf("geting new receiver failed: %s", err.Error()))
	}

	defer func() {
		closeContext, closeCancel := context.WithTimeout(ctx, 10*time.Second)
		err := receiver.Close(closeContext)
		closeCancel()
		if err != nil {
			fmt.Printf("Unable to close the receiver: %s", err)
		}
	}()

	rssData, err := a.retrieveRssDataForQueue(queueName)
	if err != nil {
		return err
	}
	numberOfMessages := len(rssData.Channel.Item)
	if numberOfMessages == 0 {
		return errors.New("no items found in queue")
	}

	for i := 0; i < numberOfMessages; i++ {
		ctxForReceive, cancelFunction := context.WithTimeout(ctx, 1*time.Second)
		msg, err := receiver.Receive(ctxForReceive)
		if ctxForReceive.Err() != nil {
			log.Println("break from context")
			break
		}
		cancelFunction()
		if err != nil {
			fmt.Printf("Unable to receive messages: %s", err)
			continue
		}

		if fmt.Sprintf("%v", msg.Properties.MessageID) == messageID {

			if err := msg.Accept(); err != nil {
				log.Printf("error trying to accept message %s, error is %s", messageID, err)
				return err
			}
			// got the message we wanted and moved it, can stop processing now
			return nil
		}
		// release the message if it wasn't the one we were looking for
		if err := msg.Release(); err != nil {
			log.Printf("error trying to release message %s", err)
		}

	}

	return nil
}

func (a *ActiveMQAdapter) DeleteMany(ctx context.Context, queueName string, messageIDs []string) []error {
	var deleteErrors []error

	session, err, closeSession := a.getSession(ctx)
	if err != nil {
		return append(deleteErrors, errors.New(fmt.Sprintf("Get new session failed: %s", err.Error())))
	}
	defer closeSession()
	var receiver *amqp.Receiver
	receiver, err = session.NewReceiver(
		amqp.LinkSourceAddress(queueName),
		amqp.LinkCredit(10),
	)
	if err != nil {
		return append(deleteErrors, errors.New(fmt.Sprintf("geting new receiver failed: %s", err.Error())))
	}

	defer func() {
		closeContext, closeCancel := context.WithTimeout(ctx, 10*time.Second)
		err := receiver.Close(closeContext)
		closeCancel()
		if err != nil {
			fmt.Printf("Unable to close the receiver: %s", err)
		}
	}()

	rssData, err := a.retrieveRssDataForQueue(queueName)
	if err != nil {
		return append(deleteErrors, err)
	}
	numberOfMessages := len(rssData.Channel.Item)
	if numberOfMessages == 0 {
		return append(deleteErrors, errors.New("no items found in queue"))
	}

	messages := make(map[string]*amqp.Message)
	for i := 0; i < numberOfMessages; i++ {
		ctxForReceive, cancelFunction := context.WithTimeout(ctx, 1*time.Second)
		msg, err := receiver.Receive(ctxForReceive)
		if ctxForReceive.Err() != nil {
			log.Println("break from context")
			cancelFunction()
			return append(deleteErrors, ctxForReceive.Err())
		}
		cancelFunction()
		if err != nil {
			fmt.Printf("Unable to receive messages: %s", err)
			return append(deleteErrors, ctxForReceive.Err())
		}

		var msgId string
		var ok bool
		if msgId, ok = msg.Properties.MessageID.(string); !ok {
			continue
		}

		messages[msgId] = msg
	}

	for _, msgId := range messageIDs {
		var msg *amqp.Message
		var ok bool
		if msg, ok = messages[msgId]; !ok {
			deleteErrors = append(deleteErrors, fmt.Errorf("Did not find message %s", msgId))
			continue
		}

		if err := msg.Accept(); err != nil {
			log.Printf("error trying to accept message %s, error is %s", msgId, err)
			continue
		}

		delete(messages, msgId)
	}

	for _, msg := range messages {
		// release the message if it wasn't the one we were looking for
		if err := msg.Release(); err != nil {
			log.Printf("error trying to release message %s", err)
		}
	}

	return deleteErrors
}

func (a *ActiveMQAdapter) GetAllQueues(ctx context.Context) ([]Queue, error) {

	httpClient := &http.Client{Timeout: time.Second * 10}

	var resp *http.Response
	var req *http.Request
	var err error
	var xmlData QueuesXMLData

	for _, brokerConsoleUrl := range a.brokerConsoleUrls {
		url := fmt.Sprintf("%s/admin/xml/queues.jsp", brokerConsoleUrl)
		log.Printf("Attempting to get queue information from %s", url)

		req, err = http.NewRequest("GET", url, nil)
		if err != nil {
			log.Printf("Unable to get a http.NewRequest for consoleURL %s, error is %s", url, err.Error())
			continue
		}

		req.SetBasicAuth(a.brokerConsoleUsr, a.brokerConsolePwd)

		resp, err = httpClient.Do(req)
		if err != nil {
			log.Printf("Error returned from this attempt was %s, url: %s", err.Error(), brokerConsoleUrl)
			continue
		}

		if resp == nil {
			log.Printf(fmt.Sprintf("Unable to retrieve info about the queue from the console consoleURL: %s", url))
			continue
		}

		if resp.StatusCode != http.StatusOK {
			log.Printf("RSS status code: %s, url: %s", resp.Status, brokerConsoleUrl)
			continue
		}

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Printf("Response body reader error: %s, url: %s", err, brokerConsoleUrl)
			continue
		}

		xmlData = QueuesXMLData{}

		err = xml.Unmarshal(body, &xmlData)
		if err != nil {
			log.Printf("XML unmarshal error: %s, url: %s", err, brokerConsoleUrl)
			continue
		}

		log.Printf("Retrieved queue information from consoleURL %s", brokerConsoleUrl)
		break
	}

	if err != nil {
		log.Printf("Unable to connect with any consoleURLs: %s", err)
		return nil, err
	}

	defer resp.Body.Close()

	queueInfoResult := []Queue{}

	for _, queue := range xmlData.Queue {
		queueInfoResult = append(queueInfoResult, Queue{
			Name: queue.Name,
			Info: map[string]string{"Size": queue.Stats.Size},
		})
	}

	return queueInfoResult, nil
}

func (a *ActiveMQAdapter) getSession(ctx context.Context) (*amqp.Session, error, func()) {
	session, err := a.client.NewSession()

	if err != nil {
		log.Printf("error attempting to start new session %v", err)
		// if err.Error() == CONNECTION_ERROR_MESSAGE {
		// 	log.Println("calling Reconnect")
		// 	go a.Reconnect(debugChannel)
		// 	return nil, errors.New(CONNECTION_LOST), nil
		// }
		return nil, errors.New(fmt.Sprintf("Get session failed, connection to client failed: %s", err.Error())), nil
	}

	closeSession := func() {
		ctx2, cancel := context.WithTimeout(ctx, 10*time.Second)
		log.Println("Closing session")
		_ = session.Close(ctx2)
		cancel()
	}

	return session, nil, closeSession
}

func (a *ActiveMQAdapter) convertMessagesToStandardMessage(
	ctx context.Context, messages []*amqp.Message) ([]structs.StandardMessage, []error) {

	var stdMessages []structs.StandardMessage

	for _, msg := range messages {

		headers := make(map[string]string)

		var body string
		getDataBody := msg.GetData()

		if getDataBody == nil {
			var ok bool
			body, ok = msg.Value.(string)
			if !ok {
				body = "<unknown body structure>"
			}
		} else {
			body = string(getDataBody)
		}

		messageId := fmt.Sprintf("%v", msg.Properties.MessageID)

		headers["Correlation ID"] = fmt.Sprintf("%v", msg.Properties.CorrelationID)
		headers["Durable"] = fmt.Sprintf("%v", msg.Header.Durable)
		headers["Priority"] = fmt.Sprintf("%v", msg.Header.Priority)
		headers["TTL"] = fmt.Sprintf("%v", msg.Header.TTL)
		headers["First Acquirer"] = fmt.Sprintf("%v", msg.Header.FirstAcquirer)
		headers["Delivery Count"] = fmt.Sprintf("%v", msg.Header.DeliveryCount)
		headers["User ID"] = fmt.Sprintf("%v", msg.Properties.UserID)
		headers["Destination"] = fmt.Sprintf("%v", msg.Properties.To)
		headers["Subject"] = fmt.Sprintf("%v", msg.Properties.Subject)
		headers["Reply To"] = fmt.Sprintf("%v", msg.Properties.ReplyTo)
		headers["Type"] = fmt.Sprintf("%v", msg.Properties.ContentType)
		headers["Group ID"] = fmt.Sprintf("%v", msg.Properties.GroupID)
		headers["Group Sequence"] = fmt.Sprintf("%v", msg.Properties.GroupSequence)

		for key, val := range msg.ApplicationProperties {
			headers[key] = fmt.Sprintf("%v", val)
		}

		for key, val := range msg.Annotations {
			keyStr := fmt.Sprintf("%v", key)
			headers[keyStr] = fmt.Sprintf("%v", val)
		}

		for key, val := range msg.DeliveryAnnotations {
			keyStr := fmt.Sprintf("%v", key)
			headers[keyStr] = fmt.Sprintf("%v", val)
		}

		stdMsg := structs.StandardMessage{
			MessageID: messageId,
			Timestamp: msg.Properties.CreationTime,
			Headers:   headers,
			Body:      body,
		}

		stdMessages = append(stdMessages, stdMsg)
	}

	return stdMessages, nil
}

// getNewSender creates a new session on the active client and then a new sender on that session
func (a *ActiveMQAdapter) getNewSender(ctx context.Context, destination string) (*amqp.Sender, error, func()) {
	session, err, closeSession := a.getSession(ctx)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("activeMQAdapter getNewSender - new session failed: %s", err.Error())), nil
	}
	var sender *amqp.Sender
	sender, err = session.NewSender(
		amqp.LinkTargetAddress(destination),
	)
	if err != nil {
		closeSession()
		return nil, errors.New(fmt.Sprintf("brokerClient getNewSender - get sender failed: %s", err.Error())), nil
	}
	log.Println("brokerClient getNewSender - sender succeeded")
	return sender, nil, closeSession
}

// Helper method to retrieve Rss data for the queue
func (a *ActiveMQAdapter) retrieveRssDataForQueue(queueName string) (*Rss, error) {
	var rssData *Rss
	var err error

	for _, brokerConsoleUrl := range a.brokerConsoleUrls {
		rssData, err = retrieveRssDataForQueue(queueName, brokerConsoleUrl, a.brokerConsoleUsr, a.brokerConsolePwd)
		if err == nil {
			break
		}
	}
	return rssData, err
}

func retrieveRssDataForQueue(queueName string, brokerConsoleUrl string, username string, password string) (*Rss, error) {

	httpClient := &http.Client{Timeout: time.Second * 10}
	var resp *http.Response

	url := fmt.Sprintf("%s/admin/queueBrowse/%s?view=rss&amp;feedType=atom_1.0", brokerConsoleUrl, queueName)
	log.Printf("attempting to get queue information from %s", url)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Printf("we were unable to get a http.NewRequest for url %s, error is %s", url, err.Error())
		return nil, errors.New(fmt.Sprintf("we were unable to get a http.NewRequest for url %s, error is %s", url, err.Error()))
	}
	req.SetBasicAuth(username, password)

	resp, err = httpClient.Do(req)
	if err != nil {
		log.Printf("error returned from this attempt was %s", err.Error())
		return nil, errors.New(fmt.Sprintf("error returned from this attempt was %s", err.Error()))
	}

	if resp.StatusCode != http.StatusOK {
		return nil, errors.New(fmt.Sprintf("invalid status when trying to retrieve from queue, status %s", resp.Status))
	}

	//if none of the urls worked and we don't have a response, we need to panic
	if resp == nil {
		return nil, errors.New(fmt.Sprintf("we were not able to retrieve info about the queue from the console urls %s", brokerConsoleUrl))
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var rssData Rss
	err = xml.Unmarshal([]byte(body), &rssData)
	if err != nil {
		fmt.Printf("error: %v", err)
		return nil, err
	}
	return &rssData, nil
}

type Rss struct {
	XMLName xml.Name `xml:"rss"`
	Text    string   `xml:",chardata"`
	Dc      string   `xml:"dc,attr"`
	Version string   `xml:"version,attr"`
	Channel struct {
		Text        string `xml:",chardata"`
		Title       string `xml:"title"`
		Link        string `xml:"link"`
		Description string `xml:"description"`
		Item        []struct {
			Text        string `xml:",chardata"`
			Title       string `xml:"title"`
			Link        string `xml:"link"`
			Description string `xml:"description"`
			PubDate     string `xml:"pubDate"`
			Guid        string `xml:"guid"`
			Date        string `xml:"date"`
		} `xml:"item"`
	} `xml:"channel"`
}
