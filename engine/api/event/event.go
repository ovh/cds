package event

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/go-gorp/gorp"
	gocache "github.com/patrickmn/go-cache"
	"github.com/rockbears/log"

	"github.com/ovh/cds/engine/api/integration"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/namesgenerator"
)

// cache with go cache
var brokersConnectionCache = gocache.New(10*time.Minute, 6*time.Hour)
var hostname, cdsname string
var brokers []Broker
var subscribers []chan<- sdk.Event

func init() {
	subscribers = make([]chan<- sdk.Event, 0)
}

// Broker event typed
type Broker interface {
	initialize(ctx context.Context, options interface{}) (Broker, error)
	sendEvent(event *sdk.Event) error
	status() string
	close(ctx context.Context)
}

func getBroker(ctx context.Context, t string, option interface{}) (Broker, error) {
	switch t {
	case "kafka":
		k := &KafkaClient{}
		return k.initialize(ctx, option)
	}
	return nil, fmt.Errorf("invalid Broker Type %s", t)
}

func getKafkaConfig(cfg sdk.IntegrationConfig) KafkaConfig {
	kafkaCfg := KafkaConfig{
		Enabled:         true,
		BrokerAddresses: cfg["broker url"].Value,
		Topic:           cfg["topic"].Value,
		MaxMessageByte:  10000000,
	}

	if _, ok := cfg["disableTLS"]; ok && cfg["disableTLS"].Value == "true" {
		kafkaCfg.DisableTLS = true
	}
	if _, ok := cfg["disableSASL"]; ok && cfg["disableSASL"].Value == "true" {
		kafkaCfg.DisableSASL = true
	} else {
		kafkaCfg.User = cfg["username"].Value
		kafkaCfg.Password = cfg["password"].Value
	}
	if _, ok := cfg["user"]; ok && cfg["user"].Value != "" {
		kafkaCfg.ClientID = cfg["user"].Value
	} else {
		kafkaCfg.ClientID = "cds"
	}
	return kafkaCfg
}

// DeleteEventIntegration delete broker connection for this event integration
func DeleteEventIntegration(eventIntegrationID int64) {
	brokerConnectionKey := strconv.FormatInt(eventIntegrationID, 10)
	brokersConnectionCache.Delete(brokerConnectionKey)
}

// ResetEventIntegration reset event integration in order to kill existing connection and add/check the new one
func ResetEventIntegration(ctx context.Context, db gorp.SqlExecutor, eventIntegrationID int64) error {
	brokerConnectionKey := strconv.FormatInt(eventIntegrationID, 10)
	brokersConnectionCache.Delete(brokerConnectionKey)
	projInt, err := integration.LoadProjectIntegrationByIDWithClearPassword(ctx, db, eventIntegrationID)
	if err != nil {
		return fmt.Errorf("cannot load project integration id %d and type event: %v", eventIntegrationID, err)
	}

	kafkaCfg := getKafkaConfig(projInt.Config)
	kafkaBroker, err := getBroker(ctx, "kafka", kafkaCfg)
	if err != nil {
		return sdk.WrapError(sdk.ErrBadBrokerConfiguration, "cannot get broker for %s and user %s : %v", projInt.Config["broker url"].Value, projInt.Config["username"].Value, err)
	}
	if err := brokersConnectionCache.Add(brokerConnectionKey, kafkaBroker, gocache.DefaultExpiration); err != nil {
		return sdk.WrapError(sdk.ErrBadBrokerConfiguration, "cannot add broker in cache for %s and user %s : %v", projInt.Config["broker url"].Value, projInt.Config["username"].Value, err)
	}
	return nil
}

// Initialize initializes event system
func Initialize(ctx context.Context, db *gorp.DbMap, cache Store) error {
	store = cache
	var err error
	hostname, err = os.Hostname()
	if err != nil {
		hostname = fmt.Sprintf("Error while getting Hostname: %v", err)
	}
	// generates an API name. api_foo_bar, only 3 first letters to have a readable status
	cdsname = "api_"
	for _, v := range strings.Split(namesgenerator.GetRandomNameCDS(0), "_") {
		if len(v) > 3 {
			cdsname += v[:3]
		} else {
			cdsname += v
		}
	}

	return nil
}

// Subscribe to CDS events
func Subscribe(ch chan<- sdk.Event) {
	subscribers = append(subscribers, ch)
}

// DequeueEvent runs in a goroutine and dequeue event from cache
func DequeueEvent(ctx context.Context, db *gorp.DbMap) {
	for {
		e := sdk.Event{}
		if err := store.DequeueWithContext(ctx, "events", 250*time.Millisecond, &e); err != nil {
			log.Error(ctx, "Event.DequeueEvent> store.DequeueWithContext err: %v", err)
			continue
		}
		if err := ctx.Err(); err != nil {
			log.Error(ctx, "Exiting event.DequeueEvent : %v", err)
			return
		}

		for _, s := range subscribers {
			s <- e
		}

		for _, eventIntegrationID := range e.EventIntegrationsID {
			brokerConnectionKey := strconv.FormatInt(eventIntegrationID, 10)
			brokerConnection, ok := brokersConnectionCache.Get(brokerConnectionKey)
			var brokerConfig KafkaConfig
			if !ok {
				projInt, err := integration.LoadProjectIntegrationByIDWithClearPassword(ctx, db, eventIntegrationID)
				if err != nil {
					log.Error(ctx, "Event.DequeueEvent> Cannot load project integration for project %s and id %d and type event: %v", e.ProjectKey, eventIntegrationID, err)
					continue
				}

				if projInt.Model.Public {
					continue
				}

				kafkaCfg := getKafkaConfig(projInt.Config)
				kafkaBroker, err := getBroker(ctx, "kafka", kafkaCfg)
				if err != nil {
					log.Error(ctx, "Event.DequeueEvent> cannot get broker for %s and user %s : %v", projInt.Config["broker url"].Value, projInt.Config["username"].Value, err)
					continue
				}
				if err := brokersConnectionCache.Add(brokerConnectionKey, kafkaBroker, gocache.DefaultExpiration); err != nil {
					log.Error(ctx, "Event.DequeueEvent> cannot add broker in cache for %s and user %s : %v", projInt.Config["broker url"].Value, projInt.Config["username"].Value, err)
					continue
				}
				brokerConnection = kafkaBroker
				brokerConfig = kafkaCfg
			}

			broker, ok := brokerConnection.(Broker)
			if !ok {
				log.Error(ctx, "cannot make cast of brokers")
				continue
			}

			// Send into external brokers
			log.Info(ctx, "sending event %q to %s", e.EventType, brokerConfig.BrokerAddresses)
			if err := broker.sendEvent(&e); err != nil {
				log.Warn(ctx, "Error while sending message [%s: %s/%s/%s/%s/%s]: %s", e.EventType, e.ProjectKey, e.WorkflowName, e.ApplicationName, e.PipelineName, e.EnvironmentName, err)
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
func Close(ctx context.Context) {
	for _, b := range brokers {
		b.close(ctx)
	}
}

// Status returns Event status
func Status(ctx context.Context) sdk.MonitoringStatusLine {
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

	return sdk.MonitoringStatusLine{Component: "Event Broker", Value: o, Status: status}
}
