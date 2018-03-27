package event

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/ovh/cds/sdk/namesgenerator"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

var hostname, cdsname string
var kafkaBroker Broker
var brokers []Broker
var subscribers []chan sdk.Event

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

func init() {
	subscribers = make([]chan sdk.Event, 0)
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

// Subscribe
func Subscribe(c chan sdk.Event) {
	subscribers = append(subscribers, c)
}

// DequeueEvent runs in a goroutine and dequeue event from cache
func DequeueEvent(c context.Context) {
	for {
		e := sdk.Event{}
		Cache.DequeueWithContext(c, "events", &e)
		if err := c.Err(); err != nil {
			log.Error("Exiting event.DequeueEvent : %v", err)
			return
		}

		// Send into internal channels
		for _, c := range subscribers {
			c <- e
		}

		// Send into external brokers
		for _, b := range brokers {
			if err := b.sendEvent(&e); err != nil {
				log.Warning("Error while sending message: %s", err)
			}
		}
	}
}

// GetHostname returns Hostname of this cds instance
func GetHostname() string {
	return hostname
}

// GetCDSName returns cdsname of this cds instance
func GetCDSName() string {
	return cdsname
}

// Close closes event system
func Close() {
	for _, b := range brokers {
		b.close()
	}
}

// Status returns Event status
func Status() sdk.MonitoringStatusLine {
	var o string
	var isAlert bool
	for _, b := range brokers {
		s := b.status()
		if !strings.Contains(s, "OK") {
			isAlert = true
		}
		o += s + " "
	}

	if o == "" {
		o = "⚠ "
	}
	status := sdk.MonitoringStatusOK
	if isAlert {
		status = sdk.MonitoringStatusAlert
	}

	return sdk.MonitoringStatusLine{Component: "Event", Value: o, Status: status}
}
