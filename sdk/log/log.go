package log

import (
	"crypto/tls"
	"fmt"
	"io/ioutil"

	log "github.com/Sirupsen/logrus"
	loghook "github.com/ovh/logrus-ovh-hook"
)

// Conf contains log configuration
type Conf struct {
	Level             string
	GraylogHost       string
	GraylogPort       string
	GraylogProtocol   string
	GraylogExtraKey   string
	GraylogExtraValue string
}

// Initialize init log level
func Initialize(conf *Conf) {
	switch conf.Level {
	case "debug":
		log.SetLevel(log.DebugLevel)
	case "info":
		log.SetLevel(log.InfoLevel)
	case "error":
		log.SetLevel(log.ErrorLevel)
	default:
		log.SetLevel(log.InfoLevel)
	}
	log.SetFormatter(&CDSFormatter{})

	if conf.GraylogHost != "" && conf.GraylogPort != "" {
		graylogcfg := &loghook.Config{
			Addr:      fmt.Sprintf("%s:%s", conf.GraylogHost, conf.GraylogPort),
			Protocol:  conf.GraylogProtocol,
			TLSConfig: &tls.Config{ServerName: conf.GraylogHost},
		}

		var extra map[string]interface{}
		if conf.GraylogExtraKey != "" && conf.GraylogExtraValue != "" {
			extra = map[string]interface{}{
				conf.GraylogExtraKey: conf.GraylogExtraValue,
			}
		}

		h, err := loghook.NewHook(graylogcfg, extra)

		if err != nil {
			log.Errorf("Error while initialize graylog hook: %s", err)
		} else {
			log.AddHook(h)
			log.SetOutput(ioutil.Discard)
		}
	}
}

// Debug prints debug log
func Debug(format string, values ...interface{}) {
	log.Debugf(format, values...)
}

// Info prints information log
func Info(format string, values ...interface{}) {
	log.Infof(format, values...)
}

// Warning prints warnings for user
func Warning(format string, values ...interface{}) {
	log.Warnf(format, values...)
}

// Critical prints error informations
func Critical(format string, values ...interface{}) {
	log.Errorf(format, values...)
}

// Fatalf prints fatal informations, then os.Exit(1)
func Fatalf(format string, values ...interface{}) {
	log.Fatalf(format, values...)
}
