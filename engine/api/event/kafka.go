package event

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Shopify/sarama"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

// KafkaClient enbeddes the Kafka connecion
type KafkaClient struct {
	options  KafkaConfig
	producer sarama.SyncProducer
}

// KafkaConfig handles all config to connect to Kafka
type KafkaConfig struct {
	Enabled         bool
	BrokerAddresses string
	User            string
	Password        string
	Topic           string
	MaxMessageByte  int
	DisableTLS      bool
	DisableSASL     bool
	ClientID        string
}

// initialize returns broker, isInit and err if
func (c *KafkaClient) initialize(ctx context.Context, options interface{}) (Broker, error) {
	conf, ok := options.(KafkaConfig)
	if !ok {
		return nil, fmt.Errorf("Invalid Kafka Initialization")
	}

	if conf.BrokerAddresses == "" ||
		conf.Topic == "" {
		return nil, fmt.Errorf("initKafka> Invalid Kafka Configuration")
	}
	c.options = conf

	if err := c.initProducer(); err != nil {
		return nil, fmt.Errorf("initKafka> Error with init sarama:%v (newSyncProducer on %s user:%s)", err, conf.BrokerAddresses, conf.User)
	}

	return c, nil
}

// close closes producer
func (c *KafkaClient) close(ctx context.Context) {
	if c.producer != nil {
		if err := c.producer.Close(); err != nil {
			log.Warning(ctx, "closeKafka> Error while closing kafka producer:%v", err)
		}
	}
}

// initProducer initializes kafka producer
func (c *KafkaClient) initProducer() error {
	var config = sarama.NewConfig()

	config.Net.TLS.Enable = !c.options.DisableTLS
	config.Net.SASL.Enable = !c.options.DisableSASL
	if config.Net.SASL.Enable {
		config.Net.SASL.User = c.options.User
		config.Net.SASL.Password = c.options.Password
	}

	config.ClientID = c.options.ClientID
	config.Producer.Return.Successes = true
	if config.Producer.MaxMessageBytes != 0 {
		config.Producer.MaxMessageBytes = c.options.MaxMessageByte
	}

	producer, err := sarama.NewSyncProducer(strings.Split(c.options.BrokerAddresses, ","), config)
	if err != nil {
		return fmt.Errorf("initKafka> Error with init sarama:%v (newSyncProducer on %s user:%s)", err, c.options.BrokerAddresses, c.options.User)
	}

	log.Debug("initKafka> Kafka used at %s on topic:%s", c.options.BrokerAddresses, c.options.Topic)
	c.producer = producer
	return nil
}

// sendOnKafkaTopic send a hook on a topic kafka
func (c *KafkaClient) sendEvent(event *sdk.Event) error {
	data, errm := json.Marshal(event)
	if errm != nil {
		return errm
	}

	msg := &sarama.ProducerMessage{Topic: c.options.Topic, Value: sarama.ByteEncoder(data)}
	if _, _, err := c.producer.SendMessage(msg); err != nil {
		return err
	}
	return nil
}

// status: here, if c is initialized, Kafka is ok
func (c *KafkaClient) status() string {
	return "Kafka OK"
}
