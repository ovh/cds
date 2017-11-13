package log

import (
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"os"
	"regexp"

	loghook "github.com/ovh/logrus-ovh-hook"
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
}

var (
	logger        Logger
	regexpNewLine = regexp.MustCompile("\\n")
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

		if conf.GraylogFieldCDSName != "" {
			extra["CDSName"] = conf.GraylogFieldCDSName
		}

		if conf.GraylogFieldCDSVersion != "" {
			extra["CDSVersion"] = conf.GraylogFieldCDSVersion
		}

		// no need to check error here
		hostname, _ := os.Hostname()
		extra["CDSHostname"] = hostname

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
	input := regexpNewLine.ReplaceAllString(fmt.Sprintf(format, values...), " ")
	if logger != nil {
		logger.Logf("[DEBUG]    " + input)
	} else {
		log.Debugf(input)
	}
}

// Info prints information log
func Info(format string, values ...interface{}) {
	input := regexpNewLine.ReplaceAllString(fmt.Sprintf(format, values...), " ")
	if logger != nil {
		logger.Logf("[INFO]    " + input)
	} else {
		log.Infof(input)
	}
}

// Warning prints warnings for user
func Warning(format string, values ...interface{}) {
	input := regexpNewLine.ReplaceAllString(fmt.Sprintf(format, values...), " ")
	if logger != nil {
		logger.Logf("[WARN]    " + input)
	} else {
		log.Warnf(input)
	}
}

// Error prints error informations
func Error(format string, values ...interface{}) {
	input := regexpNewLine.ReplaceAllString(fmt.Sprintf(format, values...), " ")
	if logger != nil {
		logger.Logf("[ERROR]    " + input)
	} else {
		log.Errorf(input)
	}
}

// Fatalf prints fatal informations, then os.Exit(1)
func Fatalf(format string, values ...interface{}) {
	input := regexpNewLine.ReplaceAllString(fmt.Sprintf(format, values...), " ")
	if logger != nil {
		logger.Logf("[FATAL]    " + input)
	} else {
		log.Fatalf(input)
	}
}
