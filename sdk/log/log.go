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

var (
	logger Logger
)

// Logger defines the logs levels used
type Logger interface {
	Logf(fmt string, values ...interface{})
}

// SetLogger override private logger reference
func SetLogger(l Logger) {
	logger = l
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
	if logger != nil {
		logger.Logf("[DEBUG]    "+format, values...)
	} else {
		log.Debugf(format, values...)
	}
}

// Info prints information log
func Info(format string, values ...interface{}) {
	if logger != nil {
		logger.Logf("[INFO]    "+format, values...)
	} else {
		log.Infof(format, values...)
	}
}

// Warning prints warnings for user
func Warning(format string, values ...interface{}) {
	if logger != nil {
		logger.Logf("[WARN]    "+format, values...)
	} else {
		log.Warnf(format, values...)
	}
}

// Critical prints error informations
func Critical(format string, values ...interface{}) {
	if logger != nil {
		logger.Logf("[ERROR]    "+format, values...)
	} else {
		log.Errorf(format, values...)
	}
}

// Fatalf prints fatal informations, then os.Exit(1)
func Fatalf(format string, values ...interface{}) {
	if logger != nil {
		logger.Logf("[FATAL]    "+format, values...)
	} else {
		log.Fatalf(format, values...)
	}
}
