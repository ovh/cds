package main

import (
	"encoding/json"
	"time"

	"github.com/mattbaird/elastigo/lib"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/event"
)

var esConn *elastigo.Conn

func consume(config Configuration, c chan<- sdk.Event) {
	log.Infof("Consuming kafka message")
	if err := event.ConsumeKafka(config.Kafka.Brokers, config.Kafka.Topic, config.Kafka.Group, config.Kafka.User, config.Kafka.Password,
		func(e sdk.Event) error {
			log.Infof("Receiving message %s", e.EventType)
			c <- e
			return nil
		},
		log.Errorf,
	); err != nil {
		log.Fatalf("Cannot consume kafka %s", err)
	}
}

func sendToES(config Configuration, c <-chan sdk.Event) {
	//Only one ES Connection
	esConn = elastigo.NewConn()

	esConn.Protocol = config.ElasticSearch.Protocol
	esConn.Domain = config.ElasticSearch.Domain
	esConn.Port = config.ElasticSearch.Port
	esConn.Username = config.ElasticSearch.Username
	esConn.Password = config.ElasticSearch.Password

	esIndex := config.ElasticSearch.Index

	for event := range c {
		jsonPayload, errM := json.Marshal(event.Payload)
		if errM != nil {
			log.Errorf("cannot marshal payload :%s", errM)
			continue
		}
		dataES := map[string]interface{}{
			"Username":  event.Username,
			"Email":     event.UserMail,
			"CDSName":   event.CDSName,
			"EventType": event.EventType,
			"Hostname":  event.Hostname,
			"Attempts":  event.Attempts,
			"Timestamp": event.Timestamp,
			"Event":     string(jsonPayload),
		}
		_, err := esConn.IndexWithParameters(esIndex, event.EventType, "0", "", 0, "", "", event.Timestamp.Format(time.RFC3339), 0, "", "", false, nil, dataES)
		time.Sleep(time.Duration(viper.GetInt("pause_es")) * time.Millisecond)
		if err != nil {
			log.Errorf("cannot index message %+v in :%s", dataES, err)
		}
	}
}
