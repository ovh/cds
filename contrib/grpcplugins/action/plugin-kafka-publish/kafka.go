package main

import (
	"strings"

	"github.com/Shopify/sarama"
)

//Client to send to kafka
func initKafkaProducer(kafka, user, password string) (sarama.SyncProducer, error) {
	c := sarama.NewConfig()
	c.Net.TLS.Enable = true
	c.Net.SASL.Enable = true
	c.Net.SASL.User = user
	c.Net.SASL.Password = password
	c.ClientID = user
	c.Producer.Return.Successes = true

	producer, err := sarama.NewSyncProducer(strings.Split(kafka, ","), c)
	if err != nil {
		return nil, err
	}
	return producer, nil
}

//Send data as a byte arrays array to kafka
func sendDataOnKafka(producer sarama.SyncProducer, topic string, data [][]byte) (int, int, error) {
	var partition int32
	var offset int64
	var err error

	for _, m := range data {
		msg := &sarama.ProducerMessage{Topic: topic, Value: sarama.ByteEncoder(m)}
		partition, offset, err = producer.SendMessage(msg)
		if err != nil {
			return 0, 0, err
		}
	}
	return int(partition), int(offset), nil
}
