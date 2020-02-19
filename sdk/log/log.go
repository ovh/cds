package log

import (
	"context"
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	log "github.com/sirupsen/logrus"

	loghook "github.com/ovh/cds/sdk/log/hook"
)

// Conf contains log configuration
type Conf struct {
	Level                      string
	GraylogHost                string
	GraylogPort                string
	GraylogProtocol            string
	GraylogExtraKey            string
	GraylogExtraValue          string
	GraylogFieldCDSServiceType string
	GraylogFieldCDSServiceName string
	GraylogFieldCDSVersion     string
	GraylogFieldCDSOS          string
	GraylogFieldCDSArch        string
	Ctx                        context.Context
}

const (
	HeaderRequestID            = "Request-ID"
	ContextLoggingRequestIDKey = "ctx-logging-request-id"
	ContextLoggingFuncKey      = "ctx-logging-func"
)

var (
	logger Logger
	hook   *loghook.Hook
)

// Logger defines the logs levels used
type Logger interface {
	Logf(fmt string, values ...interface{})
	Errorf(fmt string, values ...interface{})
	Fatalf(fmt string, values ...interface{})
}

// SetLogger replace logrus logger with custom one.
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
	case "warning":
		log.SetLevel(log.WarnLevel)
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

		extra := map[string]interface{}{}
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

		if conf.GraylogFieldCDSServiceName != "" {
			extra["CDSName"] = conf.GraylogFieldCDSServiceName
		}
		if conf.GraylogFieldCDSServiceName != "" {
			extra["CDSService"] = conf.GraylogFieldCDSServiceType
		}
		if conf.GraylogFieldCDSVersion != "" {
			extra["CDSVersion"] = conf.GraylogFieldCDSVersion
		}
		if conf.GraylogFieldCDSOS != "" {
			extra["CDSOS"] = conf.GraylogFieldCDSOS
		}
		if conf.GraylogFieldCDSArch != "" {
			extra["CDSArch"] = conf.GraylogFieldCDSArch
		}

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
		log.Info(conf.Ctx, "Draining logs")
		if hook != nil {
			hook.Flush()
		}
	}()
}

// Debug prints debug log
func Debug(format string, values ...interface{}) {
	if logger != nil {
		logger.Logf("[DEBUG] "+format, values...)
		return
	}
	log.Debugf(format, values...)
}

// InfoWithoutCtx prints information log.
func InfoWithoutCtx(format string, values ...interface{}) {
	Info(nil, format, values...)
}

// Info prints information log.
func Info(ctx context.Context, format string, values ...interface{}) {
	InfoWithFields(ctx, nil, format, values...)
}

// InfoWithFields print info log with given logrus fields.
func InfoWithFields(ctx context.Context, fields log.Fields, format string, values ...interface{}) {
	if logger != nil {
		logger.Logf("[INFO] "+format, values...)
		return
	}
	newEntry(ctx, fields).Infof(format, values...)
}

// Warning prints warnings log.
func Warning(ctx context.Context, format string, values ...interface{}) {
	WarningWithFields(ctx, nil, format, values...)
}

// WarningWithFields print warning log with given logrus fields.
func WarningWithFields(ctx context.Context, fields log.Fields, format string, values ...interface{}) {
	if logger != nil {
		logger.Logf("[WARN] "+format, values...)
		return
	}
	newEntry(ctx, fields).Warningf(format, values...)
}

// Error prints error log.
func Error(ctx context.Context, format string, values ...interface{}) {
	ErrorWithFields(ctx, nil, format, values...)
}

// ErrorWithFields print error log with given logrus fields.
func ErrorWithFields(ctx context.Context, fields log.Fields, format string, values ...interface{}) {
	if logger != nil {
		logger.Logf("[ERROR] "+format, values...)
		return
	}
	newEntry(ctx, fields).Errorf(format, values...)
}

// Fatalf prints fatal informations, then os.Exit(1)
func Fatalf(format string, values ...interface{}) {
	if logger != nil {
		logger.Logf("[FATAL] "+format, values...)
		return
	}
	log.Fatalf(format, values...)
}

func newEntry(ctx context.Context, fields log.Fields) *log.Entry {
	entry := log.NewEntry(log.StandardLogger())
	if fields != nil {
		entry = entry.WithFields(fields)
	}
	if ctx == nil {
		return entry
	}

	// Add request info if exists
	iRequestID := ctx.Value(ContextLoggingRequestIDKey)
	if iRequestID != nil {
		if requestID, ok := iRequestID.(string); ok {
			entry = entry.WithField("request_id", requestID)
		}
	}

	// If a logging func exists in context, execute it
	iFunc := ctx.Value(ContextLoggingFuncKey)
	if iFunc != nil {
		if f, ok := iFunc.(func(ctx context.Context) log.Fields); ok {
			contextFields := f(ctx)
			entry = entry.WithFields(contextFields)
		}
	}

	return entry
}
