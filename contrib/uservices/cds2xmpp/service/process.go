package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/mattn/go-xmpp"
	"github.com/mitchellh/mapstructure"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/event"
)

var conferences []string

func do() {
	event.ConsumeKafka(viper.GetString("event_kafka_broker_addresses"),
		viper.GetString("event_kafka_topic"),
		viper.GetString("event_kafka_group"),
		viper.GetString("event_kafka_user"),
		viper.GetString("event_kafka_password"),
		func(e sdk.Event) error {
			return process(e)
		},
		log.Errorf,
	)
}

func process(event sdk.Event) error {
	var eventNotif sdk.EventNotif
	log.Debugf("process> receive: type:%s", event.EventType)

	// skip all event != eventNotif
	if event.EventType != fmt.Sprintf("%T", sdk.EventNotif{}) {
		log.Debugf("process> receive: type:%s - skipped", event.EventType)
		return nil
	}

	if err := mapstructure.Decode(event.Payload, &eventNotif); err != nil {
		log.Warnf("process> Error during consumption. type:%s err:%s", event.EventType, err)
		return nil
	}

	log.Debugf("process> event:%+v", event)

	for _, destination := range eventNotif.Recipients {
		fullDestination := destination
		if !strings.Contains(destination, "@") {
			fullDestination += "@" + viper.GetString("xmpp_default_hostname")
		}
		log.Debugf("process> event send to :%s", fullDestination)

		typeXMPP := getTypeChat(fullDestination)

		if typeXMPP == typeGroupChat {
			presenceToSend := true
			for _, c := range conferences {
				if strings.HasPrefix(c, fullDestination) {
					presenceToSend = false
				}
			}

			if presenceToSend {
				log.Debugf("process> presenceToSend add :%s", fullDestination)
				conferences = append(conferences, fullDestination)
				cdsbot.sendPresencesOnConfs()
				time.Sleep(30 * time.Second)
			}
		} else if viper.GetBool("force_dot") && !strings.Contains(destination, ".") {
			continue
		}

		cdsbot.chats <- xmpp.Chat{
			Remote: fullDestination,
			Type:   typeXMPP,
			Text:   eventNotif.Subject + " " + eventNotif.Body,
		}
		cdsbot.nbXMPPSent++
	}

	return nil
}
