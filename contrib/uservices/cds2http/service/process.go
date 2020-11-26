package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient"
	"github.com/ovh/cds/sdk/event"
)

func do() {
	httpClient := cdsclient.NewHTTPClient(10*time.Second, false)
	log.Debugf("do> consume kafka: %s", viper.GetString("event_kafka_topic"))
	if err := event.ConsumeKafka(
		context.Background(),
		viper.GetString("event_kafka_version"),
		viper.GetString("event_kafka_broker_addresses"),
		viper.GetString("event_kafka_topic"),
		viper.GetString("event_kafka_group"),
		viper.GetString("event_kafka_user"),
		viper.GetString("event_kafka_password"),
		func(e sdk.Event) error {
			return process(e, httpClient)
		},
		log.Errorf,
	); err != nil {
		log.Errorf("Error on init kafka:%v", err)
	}
}

func process(event sdk.Event, client *http.Client) error {
	var eventNotif sdk.EventNotif
	log.Debugf("process> receive: type:%s", event.EventType)

	// skip all event != eventNotif
	if event.EventType != fmt.Sprintf("%T", sdk.EventNotif{}) {
		log.Debugf("process> receive: type:%s - skipped", event.EventType)
		return nil
	}

	if err := json.Unmarshal(event.Payload, &eventNotif); err != nil {
		log.Warnf("process> cannot read payload. type:%s err:%s", event.EventType, err)
		return nil
	}

	b, err := json.Marshal(eventNotif)
	if err != nil {
		return err
	}

	var body io.Reader
	if len(b) > 0 {
		body = bytes.NewBuffer(b)
	}

	req, err := http.NewRequest("POST", viper.GetString("event_remote_url"), body)
	if err != nil {
		return fmt.Errorf("process> Error during http.NewRequest: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("process> Error during client.Do: %v", err)
	}
	defer resp.Body.Close()

	response, _ := ioutil.ReadAll(resp.Body)
	log.Debugf("process> event:%+v > response body: %v", event, response)

	return nil
}
