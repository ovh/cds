package event

import (
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
}

// initialize returns broker, isInit and err if
func (c *KafkaClient) initialize(options interface{}) (Broker, error) {
	conf, ok := options.(KafkaConfig)
	if !ok {
		return nil, fmt.Errorf("Invalid Kafka Initialization")
	}

	if conf.BrokerAddresses == "" ||
		conf.User == "" ||
		conf.Password == "" ||
		conf.Topic == "" {
		return nil, fmt.Errorf("initKafka> Invalid Kafka Configuration")
	}
	c.options = conf

	if err := c.initProducer(); err != nil {
		return nil, fmt.Errorf("initKafka> Error with init sarama:%s (newSyncPoducer on %s user:%s)", err.Error(), conf.BrokerAddresses, conf.User)
	}

	return c, nil
}

// close closes producer
func (c *KafkaClient) close() {
	if c.producer != nil {
		if err := c.producer.Close(); err != nil {
			log.Warning("closeKafka> Error while closing kafka producer:%s", err.Error())
		}
	}
}

// initProducer initializes kafka producer
func (c *KafkaClient) initProducer() error {
	var config = sarama.NewConfig()
	config.Net.TLS.Enable = true
	config.Net.SASL.Enable = true
	config.Net.SASL.User = c.options.User
	config.Net.SASL.Password = c.options.Password
	config.ClientID = c.options.User
	config.Producer.Return.Successes = true
	if config.Producer.MaxMessageBytes != 0 {
		config.Producer.MaxMessageBytes = c.options.MaxMessageByte
	}

	producer, errp := sarama.NewSyncProducer(strings.Split(c.options.BrokerAddresses, ","), config)
	if errp != nil {
		return fmt.Errorf("initKafka> Error with init sarama:%s (newSyncProducer on %s user:%s)", errp.Error(), c.options.BrokerAddresses, c.options.User)
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
	if _, _, errs := c.producer.SendMessage(msg); errs != nil {
		return errs
	}
	return nil
}

// status: here, if c is initialized, Kafka is ok
func (c *KafkaClient) status() string {
	return "Kafka OK"
}
