package event

import (
	"encoding/json"
	"fmt"

	"github.com/Shopify/sarama"
	"gopkg.in/bsm/sarama-cluster.v2"

	"github.com/ovh/cds/sdk"
)

// ConsumeKafka consume CDS Event from a kafka topic
func ConsumeKafka(addr, topic, group, user, password string, ProcessEventFunc func(sdk.Event) error, ErrorLogFunc func(string, ...interface{})) error {

	var config = sarama.NewConfig()
	config.Net.TLS.Enable = true
	config.Net.SASL.Enable = true
	config.Net.SASL.User = user
	config.Net.SASL.Password = password
	config.Version = sarama.V0_10_0_1

	config.ClientID = user

	clusterConfig := cluster.NewConfig()
	clusterConfig.Config = *config
	clusterConfig.Consumer.Return.Errors = true

	var errConsumer error
	consumer, errConsumer := cluster.NewConsumer(
		[]string{addr},
		group,
		[]string{topic},
		clusterConfig)

	if errConsumer != nil {
		return fmt.Errorf("Error creating consumer: %s", errConsumer)
	}

	// Consume errors
	go func() {
		for err := range consumer.Errors() {
			ErrorLogFunc("Error during consumption: %s", err)
		}
	}()

	// Consume message
	for msg := range consumer.Messages() {
		var event sdk.Event
		json.Unmarshal(msg.Value, &event)
		if err := ProcessEventFunc(event); err != nil {
			ErrorLogFunc("Error on ProcessEventFunc:%s", err)
		} else {
			consumer.MarkOffset(msg, "delivered")
		}
	}
	return nil
}
