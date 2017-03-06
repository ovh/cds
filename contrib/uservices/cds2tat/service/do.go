package main

import (
	"net/http"

	log "github.com/Sirupsen/logrus"
	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/event"
)

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

func check(ctx *gin.Context) {
	if _, err := getClient().UserMe(); err != nil {
		ctx.JSON(http.StatusOK, gin.H{"result": gin.H{"TAT": "KO"}})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"result": "OK"})
}
