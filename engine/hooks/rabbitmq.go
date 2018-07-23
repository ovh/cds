package hooks

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/fsamin/go-dump"
	"github.com/streadway/amqp"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

type rabbitMQConsumer struct {
	conn    *amqp.Connection
	channel *amqp.Channel
	tag     string
	done    chan error
}

func (s *Service) startRabbitMQHook(t *sdk.Task) error {
	projectKey := t.Config[sdk.HookConfigProject].Value
	platformName := t.Config[sdk.HookModelPlatform].Value
	pf, err := s.Client.ProjectPlatformGet(projectKey, platformName, true)
	if err != nil {
		_ = s.stopTask(t)
		return sdk.WrapError(err, "startTask> Cannot get rabbitMQ configuration for %s/%s", projectKey, platformName)
	}

	password := pf.Config["password"].Value
	username := pf.Config["username"].Value
	uri := fmt.Sprintf("amqp://%s:%s@%s", username, password, pf.Config["uri"].Value)

	consumer, err := newConsumer(
		uri,
		t.Config[sdk.RabbitMQHookModelExchangeName].Value,
		t.Config[sdk.RabbitMQHookModelExchangeType].Value,
		t.Config[sdk.RabbitMQHookModelQueue].Value,
		t.Config[sdk.RabbitMQHookModelBindingKey].Value,
		t.Config[sdk.RabbitMQHookModelConsumerTag].Value,
	)
	if err != nil {
		_ = s.stopTask(t)
		return fmt.Errorf("startRabbitMQHook>Error creating consumer: (%s %s %+v): %v", pf.Config["uri"].Value, username, t.Config, err)
	}

	deliveries, errConsume := consumer.channel.Consume(
		t.Config[sdk.RabbitMQHookModelQueue].Value, // name
		consumer.tag,                               // consumerTag,
		false,                                      // noAck
		false,                                      // exclusive
		false,                                      // noLocal
		false,                                      // noWait
		nil,                                        // arguments
	)
	if errConsume != nil {
		_ = s.stopTask(t)
		return fmt.Errorf("startRabbitMQHook> Queue Consume: %s", errConsume)
	}

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigs
		log.Info("RabbitMQ> shutdown")
		_ = consumer.Shutdown()
	}()

	go func() {
		for d := range deliveries {
			_ = d.Ack(false)
			exec := sdk.TaskExecution{
				ProcessingTimestamp: time.Now().UnixNano(),
				Status:              TaskExecutionDoing,
				Config:              t.Config,
				Type:                TypeRabbitMQ,
				UUID:                t.UUID,
				Timestamp:           time.Now().UnixNano(),
				RabbitMQ:            &sdk.RabbitMQTaskExecution{Message: d.Body},
			}
			s.Dao.SaveTaskExecution(&exec)
			s.Dao.EnqueueTaskExecution(&exec)
		}
		consumer.done <- nil
	}()

	return nil
}

func (s *Service) doRabbitMQTaskExecution(t *sdk.TaskExecution) (*sdk.WorkflowNodeRunHookEvent, error) {
	log.Debug("Hooks> Processing rabbitMQ %s %s", t.UUID, t.Type)

	// Prepare a struct to send to CDS API
	h := sdk.WorkflowNodeRunHookEvent{
		WorkflowNodeHookUUID: t.UUID,
		Payload:              map[string]string{},
	}

	var bodyJSON interface{}

	//Try to parse the body as an array
	bodyJSONArray := []interface{}{}
	if err := json.Unmarshal(t.RabbitMQ.Message, &bodyJSONArray); err != nil {
		//Try to parse the body as a map
		bodyJSONMap := map[string]interface{}{}
		if err2 := json.Unmarshal(t.RabbitMQ.Message, &bodyJSONMap); err2 == nil {
			bodyJSON = bodyJSONMap
		}
	} else {
		bodyJSON = bodyJSONArray
	}

	//Go Dump
	e := dump.NewDefaultEncoder(new(bytes.Buffer))
	e.Formatters = []dump.KeyFormatterFunc{dump.WithDefaultLowerCaseFormatter()}
	e.ExtraFields.DetailedMap = false
	e.ExtraFields.DetailedStruct = false
	e.ExtraFields.DeepJSON = true
	e.ExtraFields.Len = false
	e.ExtraFields.Type = false
	m, err := e.ToStringMap(bodyJSON)
	if err != nil {
		return nil, sdk.WrapError(err, "Hooks.doRabbitMQTaskExecution> Unable to dump body %s", t.WebHook.RequestBody)
	}
	h.Payload = m

	return &h, nil
}

func newConsumer(amqpURI, exchange, exchangeType, queueName, key, ctag string) (*rabbitMQConsumer, error) {
	c := &rabbitMQConsumer{
		conn:    nil,
		channel: nil,
		tag:     ctag,
		done:    make(chan error),
	}

	var err error

	c.conn, err = amqp.Dial(amqpURI)
	if err != nil {
		return nil, fmt.Errorf("Dial: %s", err)
	}

	go func() {
		fmt.Printf("closing: %s", <-c.conn.NotifyClose(make(chan *amqp.Error)))
	}()

	c.channel, err = c.conn.Channel()
	if err != nil {
		return nil, fmt.Errorf("Channel: %s", err)
	}

	if err = c.channel.ExchangeDeclare(
		exchange,     // name of the exchange
		exchangeType, // type
		true,         // durable
		false,        // delete when complete
		false,        // internal
		false,        // noWait
		nil,          // arguments
	); err != nil {
		return nil, fmt.Errorf("Exchange Declare: %s", err)
	}

	queue, err := c.channel.QueueDeclare(
		queueName, // name of the queue
		true,      // durable
		false,     // delete when unused
		false,     // exclusive
		false,     // noWait
		nil,       // arguments
	)
	if err != nil {
		return nil, fmt.Errorf("Queue Declare: %s", err)
	}

	if err = c.channel.QueueBind(
		queue.Name, // name of the queue
		key,        // bindingKey
		exchange,   // sourceExchange
		false,      // noWait
		nil,        // arguments
	); err != nil {
		return nil, fmt.Errorf("Queue Bind: %s", err)
	}

	return c, nil
}

func (c *rabbitMQConsumer) Shutdown() error {
	// will close() the deliveries channel
	if err := c.channel.Cancel(c.tag, true); err != nil {
		return fmt.Errorf("Consumer cancel failed: %s", err)
	}

	if err := c.conn.Close(); err != nil {
		return fmt.Errorf("AMQP connection close error: %s", err)
	}

	log.Info("RabbitMQ> Shutdown> Wait for handle to exit...")
	// wait for handle() to exit
	return <-c.done
}
