package main

import (
	"github.com/ovh/tat"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

var instance *tat.Client

// getClient initializes client on tat engine
func getClient() *tat.Client {
	if instance != nil {
		return instance
	}

	tc, err := tat.NewClient(tat.Options{
		URL:      viper.GetString("url_tat_engine"),
		Username: viper.GetString("username_tat_engine"),
		Password: viper.GetString("password_tat_engine"),
		Referer:  "tatexamplecron.v." + VERSION,
	})

	if err != nil {
		log.Errorf("Error while create new Tat Client:%s", err)
	}

	tat.DebugLogFunc = log.Debugf
	tat.ErrorLogFunc = log.Warnf

	instance = tc
	return instance
}
