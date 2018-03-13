package log

import (
	"context"
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"os"
	"runtime"
	"strings"

	loghook "github.com/ovh/cds/sdk/log/hook"
	log "github.com/sirupsen/logrus"
)

// Conf contains log configuration
type Conf struct {
	Level                  string
	GraylogHost            string
	GraylogPort            string
	GraylogProtocol        string
	GraylogExtraKey        string
	GraylogExtraValue      string
	GraylogFieldCDSName    string
	GraylogFieldCDSVersion string
	Ctx                    context.Context
}

var (
	logger Logger
	hook   *loghook.Hook
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
			keys := strings.Split(conf.GraylogExtraKey, ",")
			values := strings.Split(conf.GraylogExtraValue, ",")
			if len(keys) != len(values) {
				log.Errorf("Error while initialize log: extraKey (len:%d) does not have same corresponding number of values on extraValue (len:%d)", len(keys), len(values))
			} else {
				for i := range keys {
					extra[keys[i]] = values[i]
				}
			}
		}

		if conf.GraylogFieldCDSName != "" {
			extra["CDSName"] = conf.GraylogFieldCDSName
		}
		if conf.GraylogFieldCDSVersion != "" {
			extra["CDSVersion"] = conf.GraylogFieldCDSVersion
		}

		extra["CDSOS"] = runtime.GOOS
		extra["CDSArch"] = runtime.GOARCH

		// no need to check error here
		hostname, _ := os.Hostname()
		extra["CDSHostname"] = hostname

		var errhook error
		hook, errhook = loghook.NewHook(graylogcfg, extra)

		if errhook != nil {
			log.Errorf("Error while initialize graylog hook: %v", errhook)
		} else {
			log.AddHook(hook)
			log.SetOutput(ioutil.Discard)
		}
	}

	if conf.Ctx == nil {
		conf.Ctx = context.Background()
	}
	go func() {
		<-conf.Ctx.Done()
		log.Info("Draining logs")
		if hook != nil {
			hook.Flush()
		}
	}()
}

// Debug prints debug log
func Debug(format string, values ...interface{}) {
	input := strings.Replace(fmt.Sprintf(format, values...), "\\n", " ", -1)
	if logger != nil {
		logger.Logf("[DEBUG]    " + input)
	} else {
		log.Debugf(input)
	}
}

// Info prints information log
func Info(format string, values ...interface{}) {
	input := strings.Replace(fmt.Sprintf(format, values...), "\\n", " ", -1)
	if logger != nil {
		logger.Logf("[INFO]    " + input)
	} else {
		log.Infof(input)
	}
}

// Warning prints warnings for user
func Warning(format string, values ...interface{}) {
	input := strings.Replace(fmt.Sprintf(format, values...), "\\n", " ", -1)
	if logger != nil {
		logger.Logf("[WARN]    " + input)
	} else {
		log.Warnf(input)
	}
}

// Error prints error informations
func Error(format string, values ...interface{}) {
	input := strings.Replace(fmt.Sprintf(format, values...), "\\n", " ", -1)
	if logger != nil {
		logger.Logf("[ERROR]    " + input)
	} else {
		log.Errorf(input)
	}
}

// Fatalf prints fatal informations, then os.Exit(1)
func Fatalf(format string, values ...interface{}) {
	input := strings.Replace(fmt.Sprintf(format, values...), "\\n", " ", -1)
	if logger != nil {
		logger.Logf("[FATAL]    " + input)
	} else {
		log.Fatalf(input)
	}
}
