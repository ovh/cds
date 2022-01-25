package event

import (
	"context"
	"reflect"
	"strings"

	"github.com/Shopify/sarama"
	"github.com/pkg/errors"

	"github.com/ovh/cds/sdk"
)

// KafkaConfig handles all config to connect to Kafka
type KafkaConfig struct {
	Enabled         bool   `toml:"enabled" json:"-" default:"false" mapstructure:"enabled"`
	BrokerAddresses string `toml:"broker" json:"-"  mapstructure:"broker"`
	User            string `toml:"user" json:"-" mapstructure:"user"`
	Password        string `toml:"password" json:"-" mapstructure:"password"`
	Version         string `toml:"version" json:"-" mapstructure:"version"`
	Topic           string `toml:"topic" json:"-" mapstructure:"topic"`
	MaxMessageByte  int    `toml:"maxMessageByte" json:"-" mapstructure:"maxMessageByte"`
	DisableTLS      bool   `toml:"disableTLS" json:"-" mapstructure:"disableTLS"`
	DisableSASL     bool   `toml:"disableSASL" json:"-" mapstructure:"disableSASL"`
	ClientID        string `toml:"clientID" json:"-" mapstructure:"clientID"`
}

type KafkaConsumerConfig struct {
	KafkaConfig
	ConsumerGroup string `toml:"consumerGroup" json:"-" mapstructure:"consumerGroup"`
}

// ConsumeKafka consume CDS Event from a kafka topic
func ConsumeKafka(ctx context.Context,
	kafkaConfig KafkaConsumerConfig,
	messageType interface{},
	processEventFunc func(interface{}) error, errorLogFunc func(string, ...interface{}),
) error {
	var config = sarama.NewConfig()
	config.Net.TLS.Enable = !kafkaConfig.DisableTLS
	config.Net.SASL.Enable = !kafkaConfig.DisableSASL
	config.Net.SASL.User = kafkaConfig.User
	config.Net.SASL.Password = kafkaConfig.Password
	config.ClientID = kafkaConfig.User
	config.Consumer.Return.Errors = true

	if config.ClientID == "" {
		config.ClientID = "cds"
	}

	if kafkaConfig.Version != "" {
		kafkaVersion, err := sarama.ParseKafkaVersion(kafkaConfig.Version)
		if err != nil {
			return errors.Wrapf(err, "error parsing Kafka version %v", kafkaConfig.Version)
		}
		config.Version = kafkaVersion
	} else {
		config.Version = sarama.V0_10_2_0
	}

	consumerGroup, err := sarama.NewConsumerGroup(strings.Split(kafkaConfig.BrokerAddresses, ","), kafkaConfig.ConsumerGroup, config)
	if err != nil {
		return errors.Wrapf(err, "error creating consumer")
	}

	// Track errors
	go func() {
		for err := range consumerGroup.Errors() {
			errorLogFunc("group.Errors:%s", err)
		}
	}()

	h := handler{
		messageType:      messageType,
		processEventFunc: processEventFunc,
		errorLogFunc:     errorLogFunc,
	}
	go func() {
		for {
			if err := consumerGroup.Consume(context.Background(), []string{kafkaConfig.Topic}, &h); err != nil {
				errorLogFunc("ProcessEventFunc:%s", err)
			}
		}
	}()
	return nil
}

// handler represents a Sarama consumer group consumer
type handler struct {
	messageType      interface{}
	processEventFunc func(i interface{}) error
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
		var event = reflect.New(reflect.TypeOf(h.messageType)).Interface()

		sdk.JSONUnmarshal(message.Value, event)
		if err := h.processEventFunc(event); err != nil {
			h.errorLogFunc("ProcessEventFunc:%s", err)
		}
		session.MarkMessage(message, "delivered")
	}
	return nil
}
