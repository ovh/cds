package log

import (
	log "github.com/Sirupsen/logrus"
)

// Initialize init log level
func Initialize(level string) {
	switch level {
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
