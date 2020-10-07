package event

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/Shopify/sarama"

	"github.com/ovh/cds/sdk"
)

// ConsumeKafka consume CDS Event from a kafka topic
func ConsumeKafka(ctx context.Context, kafkaVersion, addr, topic, group, user, password string, processEventFunc func(sdk.Event) error, errorLogFunc func(string, ...interface{})) error {
	var config = sarama.NewConfig()
	config.Net.TLS.Enable = true
	config.Net.SASL.Enable = true
	config.Net.SASL.User = user
	config.Net.SASL.Password = password
	config.ClientID = user
	config.Consumer.Return.Errors = true

	if kafkaVersion != "" {
		kafkaVersion, err := sarama.ParseKafkaVersion(kafkaVersion)
		if err != nil {
			return fmt.Errorf("error parsing Kafka version %v err:%s", kafkaVersion, err)
		}
		config.Version = kafkaVersion
	} else {
		config.Version = sarama.V0_10_2_0
	}

	consumerGroup, err := sarama.NewConsumerGroup([]string{addr}, group, config)
	if err != nil {
		return fmt.Errorf("Error creating consumer: %s", err)
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
			if err := consumerGroup.Consume(context.Background(), []string{topic}, &h); err != nil {
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
		json.Unmarshal(message.Value, &event)
		if err := h.processEventFunc(event); err != nil {
			h.errorLogFunc("Error on ProcessEventFunc:%s", err)
		}
		session.MarkMessage(message, "delivered")
	}
	return nil
}
