package event

import (
	"fmt"
	"os"

	"github.com/docker/docker/pkg/namesgenerator"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/sdk/log"
	"github.com/ovh/cds/sdk"
)

var hostname, cdsname string
var kafkaBroker Broker
var brokers []Broker

// Broker event typed
type Broker interface {
	initialize(options interface{}) (Broker, error)
	sendEvent(event *sdk.Event) error
	status() string
	close()
}

func getBroker(t string, option interface{}) (Broker, error) {
	switch t {
	case "kafka":
		k := &KafkaClient{}
		return k.initialize(option)
	}
	return nil, fmt.Errorf("Invalid Broker Type %s", t)
}

// Initialize initializes event system
func Initialize(k KafkaConfig) error {
	var err error
	hostname, err = os.Hostname()
	if err != nil {
		hostname = fmt.Sprintf("Error while getting Hostname: %s", err.Error())
	}
	cdsname = namesgenerator.GetRandomName(0)

	brokers = []Broker{}
	if k.Enabled {
		var errk error
		kafkaBroker, errk = getBroker("kafka", k)
		if errk != nil {
			return errk
		}
		brokers = append(brokers, kafkaBroker)
	}
	return nil
}

// DequeueEvent runs in a goroutine and dequeue event from cache
func DequeueEvent() {
	for {
		e := sdk.Event{}
		cache.Dequeue("events", &e)
		for _, b := range brokers {
			if err := b.sendEvent(&e); err != nil {
				log.Warning("Error while sending message: %s", err)
			}
		}
	}
}

// Close closes event system
func Close() {
	for _, b := range brokers {
		b.close()
	}
}

// Status returns Event status
func Status() string {
	o := ""
	for _, b := range brokers {
		o += b.status() + " "
	}

	if o == "" {
		o = "⚠ "
	}

	return o
}
