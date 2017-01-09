package event

import (
	"fmt"
	"os"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/log"
	"github.com/ovh/cds/sdk"

	"github.com/docker/docker/pkg/namesgenerator"
	"github.com/spf13/viper"
)

var hostname, cdsname string

// Routine initializes and run event routine dequeue
func Routine() {
	var err error
	hostname, err = os.Hostname()
	if err != nil {
		hostname = fmt.Sprintf("Error while getting Hostname: %s", err.Error())
	}
	cdsname = namesgenerator.GetRandomName(0)

	withKafka := kafkaRoutine()

	for {
		e := sdk.Event{}
		cache.Dequeue("events", &e)
		// send to kafka queue if configured
		if withKafka {
			if err := sendOnKafkaTopic(producer, viper.GetString("event_kafka_topic"), &e, log.Debug); err != nil {
				log.Warning("Error while send message on kafka: %s", err)
			}
		}
	}
}
