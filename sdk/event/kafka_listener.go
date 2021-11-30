package event

import (
	"context"
	"strings"

	"github.com/Shopify/sarama"
	"github.com/pkg/errors"

	"github.com/ovh/cds/sdk"
)

type KafkaConfig struct {
	Version       string `json:"version"`
	User          string `json:"user"`
	Password      string `json:"password"`
	Topic         string `json:"topic"`
	Broker        string `json:"broker"`
	ConsumerGroup string `json:"consumerGroup"`
}

// ConsumeKafka consume CDS Event from a kafka topic
func ConsumeKafka(ctx context.Context, kafkaConfig KafkaConfig, processEventFunc func(sdk.Event) error, errorLogFunc func(string, ...interface{})) error {
	var config = sarama.NewConfig()
	config.Net.TLS.Enable = true
	config.Net.SASL.Enable = true
	config.Net.SASL.User = kafkaConfig.User
	config.Net.SASL.Password = kafkaConfig.Password
	config.ClientID = kafkaConfig.User
	config.Consumer.Return.Errors = true

	if kafkaConfig.Version != "" {
		kafkaVersion, err := sarama.ParseKafkaVersion(kafkaConfig.Version)
		if err != nil {
			return errors.Wrapf(err, "error parsing Kafka version %v", kafkaConfig.Version)
		}
		config.Version = kafkaVersion
	} else {
		config.Version = sarama.V0_10_2_0
	}

	consumerGroup, err := sarama.NewConsumerGroup(strings.Split(kafkaConfig.Broker, ","), kafkaConfig.ConsumerGroup, config)
	if err != nil {
		return errors.Wrapf(err, "error creating consumer")
	}

	// Track errors
	go func() {
		for err := range consumerGroup.Errors() {
			errorLogFunc("Error on group.Errors:%s", err)
		}
	}()

	h := handler{
		processEventFunc: processEventFunc,
		errorLogFunc:     errorLogFunc,
	}
	go func() {
		for {
			if err := consumerGroup.Consume(context.Background(), []string{kafkaConfig.Topic}, &h); err != nil {
				errorLogFunc("Error on ProcessEventFunc:%s", err)
			}
		}
	}()
	return nil
}

// handler represents a Sarama consumer group consumer
type handler struct {
	processEventFunc func(sdk.Event) error
	errorLogFunc     func(string, ...interface{})
}

// Setup is run at the beginning of a new session, before ConsumeClaim
func (h *handler) Setup(sarama.ConsumerGroupSession) error {
	return nil
}

// Cleanup is run at the end of a session, once all ConsumeClaim goroutines have exited
func (h *handler) Cleanup(sarama.ConsumerGroupSession) error {
	return nil
}

// ConsumeClaim must start a consumer loop of ConsumerGroupClaim's Messages().
func (h *handler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for message := range claim.Messages() {
		var event sdk.Event
		sdk.JSONUnmarshal(message.Value, &event)
		if err := h.processEventFunc(event); err != nil {
			h.errorLogFunc("Error on ProcessEventFunc:%s", err)
		}
		session.MarkMessage(message, "delivered")
	}
	return nil
}
