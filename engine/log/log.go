package log

import (
	log "github.com/Sirupsen/logrus"
	"github.com/spf13/viper"
)

func init() {
	switch viper.GetString("log_level") {
	case "debug":
		log.SetLevel(log.DebugLevel)
	case "info":
		log.SetLevel(log.InfoLevel)
	case "error":
		log.SetLevel(log.ErrorLevel)
	default:
		log.SetLevel(log.InfoLevel)
	}
	log.SetFormatter(&TextFormatter{})
}

// Debug prints debug log
func Debug(format string, values ...interface{}) {
	if values != nil {
		log.Debugf(format, values)
	} else {
		log.Debug(format, values)
	}
}

// Info prints information log
func Info(format string, values ...interface{}) {
	if values != nil {
		log.Infof(format, values)
	} else {
		log.Info(format, values)
	}
}

// Warning prints warnings for user
func Warning(format string, values ...interface{}) {
	if values != nil {
		log.Warnf(format, values)
	} else {
		log.Warn(format)
	}
}

// Critical prints error informations
func Critical(format string, values ...interface{}) {
	if values != nil {
		log.Errorf(format, values)
	} else {
		log.Error(format, values)
	}
}

// Fatalf prints fatal informations, then os.Exit(1)
func Fatalf(format string, values ...interface{}) {
	if values != nil {
		log.Fatalf(format, values)
	} else {
		log.Fatal(format, values)
	}
}
