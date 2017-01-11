package event

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Shopify/sarama"
	"github.com/spf13/viper"

	"github.com/ovh/cds/engine/log"
	"github.com/ovh/cds/sdk"
)

var producer sarama.SyncProducer

// Close close event system
func Close() {
	closeKafka()
}

func kafkaRoutine() bool {
	if viper.GetString("event_kafka_broker_addresses") == "" ||
		viper.GetString("event_kafka_user") == "" ||
		viper.GetString("event_kafka_password") == "" ||
		viper.GetString("event_kafka_topic") == "" {
		log.Debug("initKafka> No Kafka configured")
		return false
	}

	var errI error
	producer, errI = initProducer(
		viper.GetString("event_kafka_broker_addresses"),
		viper.GetString("event_kafka_user"),
		viper.GetString("event_kafka_password"),
		viper.GetString("event_kafka_topic"),
		log.Info)

	if errI != nil {
		log.Warning("initKafka> Error with init sarama:%s (newSyncProducer on %s user:%s)", errI.Error(), viper.GetString("event_kafka_broker_addresses"), viper.GetString("event_kafka_user"))
		return false
	}

	return true
}

// closeKafka closes producer
func closeKafka() {
	if producer != nil {
		if err := producer.Close(); err != nil {
			log.Warning("closeKafka> Error while closing kafka producer:%s", err.Error())
		}
	}
}

// initProducer initializes kafka producer
// producer could be nil
func initProducer(brokerAddresses, user, password, topic string, InfoLogFunc func(string, ...interface{})) (sarama.SyncProducer, error) {
	var config = sarama.NewConfig()
	config.Net.TLS.Enable = true
	config.Net.SASL.Enable = true
	config.Net.SASL.User = user
	config.Net.SASL.Password = password
	config.ClientID = user
	config.Producer.Return.Successes = true

	producer, errp := sarama.NewSyncProducer(strings.Split(brokerAddresses, ","), config)
	if errp != nil {
		return nil, fmt.Errorf("initKafka> Error with init sarama:%s (newSyncProducer on %s user:%s)", errp.Error(), brokerAddresses, user)
	}

	InfoLogFunc("initKafka> Kafka used at %s on topic:%s", brokerAddresses, topic)
	return producer, nil
}

// sendOnKafkaTopic send a hook on a topic kafka
func sendOnKafkaTopic(producer sarama.SyncProducer, topic string, event *sdk.Event, DebugLogFunc func(string, ...interface{})) error {
	data, errm := json.Marshal(event)
	if errm != nil {
		return errm
	}

	msg := &sarama.ProducerMessage{Topic: topic, Value: sarama.ByteEncoder(data)}
	partition, offset, errs := producer.SendMessage(msg)
	if errs != nil {
		return errs
	}
	DebugLogFunc("Event %+v sent to topic %s partition %d offset %d", event, topic, partition, offset)
	return nil
}
