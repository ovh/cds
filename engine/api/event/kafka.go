package event

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Shopify/sarama"
	"github.com/spf13/viper"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/log"
	"github.com/ovh/cds/sdk"
)

var producer sarama.SyncProducer
var hookKafkaEnabled bool
var topic string

// Routine initializes and run event routine dequeue
func Routine() {
	kafkaRoutine()
}

// Close close event system
func Close() {
	closeKafka()
}

func kafkaRoutine() {
	if viper.GetString("event_kafka_broker_addresses") == "" ||
		viper.GetString("event_kafka_user") == "" ||
		viper.GetString("event_kafka_password") == "" ||
		viper.GetString("event_kafka_topic") == "" {
		log.Debug("initKafka> No Kafka configured")
		return
	}

	var config = sarama.NewConfig()
	config.Net.TLS.Enable = true
	config.Net.SASL.Enable = true
	config.Net.SASL.User = viper.GetString("event_kafka_user")
	config.Net.SASL.Password = viper.GetString("event_kafka_password")
	config.ClientID = viper.GetString("event_kafka_user")
	config.Producer.Return.Successes = true

	topic = viper.GetString("event_kafka_topic")

	var err error
	producer, err = sarama.NewSyncProducer(strings.Split(viper.GetString("event_kafka_broker_addresses"), ","), config)
	if err != nil {
		log.Warning("initKafka> Error with init sarama:%s (newSyncProducer on %s user:%s)", err.Error(), viper.GetString("event_kafka_broker_addresses"), viper.GetString("event_kafka_user"))
	} else {
		hookKafkaEnabled = true
		log.Debug("initKafka> Kafka used at %s on topic:%s", viper.GetString("event_kafka_broker_addresses"), topic)
	}

	for {
		e := sdk.Event{}
		cache.Dequeue("events", &e)
		if e.EventType != "" {
			if err := sendOnKafkaTopic(&e); err != nil {
				log.Warning("Error while send message on kafka: %s", err)
			}
		}
	}
}

// closeKafka closes producer
func closeKafka() {
	if producer != nil {
		if err := producer.Close(); err != nil {
			log.Warning("closeKafka> Error while closing kafka producer:%s", err.Error())
		}
	}
}

// sendOnKafkaTopic send a hook on a topic kafka
func sendOnKafkaTopic(event *sdk.Event) error {
	if !hookKafkaEnabled {
		return fmt.Errorf("sendOnKafkaTopic: Kafka not initialized")
	}

	data, err := json.Marshal(event)
	if err != nil {
		return err
	}

	msg := &sarama.ProducerMessage{Topic: topic, Value: sarama.ByteEncoder(data)}
	partition, offset, err := producer.SendMessage(msg)
	if err != nil {
		return err
	}
	//log.Debug("Event %+v sent to topic %s partition %d offset %d", event, topic, partition, offset)
	log.Debug("Event sent to topic %s partition %d offset %d", topic, partition, offset)
	return nil
}
