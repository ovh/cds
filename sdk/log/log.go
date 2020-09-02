package log

import (
	"context"
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"testing"

	log "github.com/sirupsen/logrus"

	"github.com/ovh/cds/sdk/log/hook"
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
}

const (
	HeaderRequestID            = "Request-ID"
	ContextLoggingRequestIDKey = "ctx-logging-request-id"
	ContextLoggingFuncKey      = "ctx-logging-func"

	ExtraFieldSignature = "Signature"
	ExtraFieldLine      = "Line"
	ExtraFieldJobStatus = "JobStatus"
)

var (
	logger      Logger
	graylogHook *hook.Hook
)

// Logger defines the logs levels used
type Logger interface {
	Logf(fmt string, values ...interface{})
	Errorf(fmt string, values ...interface{})
	Fatalf(fmt string, values ...interface{})
}

type TestingLogger struct {
	t *testing.T
}

var _ Logger = new(TestingLogger)

func (t *TestingLogger) isDone() bool {
	return t.t.Failed() || t.t.Skipped()
}

func (t *TestingLogger) Logf(fmt string, values ...interface{}) {
	if !t.isDone() {
		t.t.Logf(fmt, values...)
	}
}
func (t *TestingLogger) Errorf(fmt string, values ...interface{}) {
	if !t.isDone() {
		t.t.Errorf(fmt, values...)
	}
}
func (t *TestingLogger) Fatalf(fmt string, values ...interface{}) {
	if !t.isDone() {
		t.t.Fatalf(fmt, values...)
	}
}

// SetLogger replace logrus logger with custom one.
func SetLogger(l Logger) {
	t, isTesting := l.(*testing.T)
	if isTesting {
		logger = &TestingLogger{t: t}
		return
	}
	logger = l
}

func logWithLogger(level string, fields log.Fields, format string, values ...interface{}) {
	var fString string
	for k, m := range fields {
		if k != "stack_trace" {
			fString = fmt.Sprintf("%s %s:%v", fString, k, m)
		}
	}
	logger.Logf("["+level+"] "+format+fString, values...)
	if fields != nil {
		if v, ok := fields["stack_trace"]; ok {
			logger.Logf("%s", v)
		}
	}
}

// Initialize init log level
func Initialize(ctx context.Context, conf *Conf) {
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
		graylogcfg := &hook.Config{
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
		graylogHook, errhook = hook.NewHook(ctx, graylogcfg, extra)

		if errhook != nil {
			log.Errorf("Error while initialize graylog hook: %v", errhook)
		} else {
			log.AddHook(graylogHook)
			log.SetOutput(ioutil.Discard)
		}
	}

	go func() {
		<-ctx.Done()
		Info(ctx, "Draining logs...")
		if graylogHook != nil {
			graylogHook.Flush()
		}
	}()
}

// Debug prints debug log
func Debug(format string, values ...interface{}) {
	if logger != nil {
		logWithLogger("DEBUG", nil, format, values...)
		return
	}
	log.Debugf(format, values...)
}

// InfoWithoutCtx prints information log.
func InfoWithoutCtx(format string, values ...interface{}) {
	Info(context.Background(), format, values...)
}

// Info prints information log.
func Info(ctx context.Context, format string, values ...interface{}) {
	InfoWithFields(ctx, nil, format, values...)
}

// InfoWithFields print info log with given logrus fields.
func InfoWithFields(ctx context.Context, fields log.Fields, format string, values ...interface{}) {
	if logger != nil {
		logWithLogger("INFO", fields, format, values...)
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
		logWithLogger("WARN", fields, format, values...)
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
		logWithLogger("ERROR", fields, format, values...)
		return
	}
	newEntry(ctx, fields).Errorf(format, values...)
}

// Fatalf prints fatal informations, then os.Exit(1)
func Fatalf(format string, values ...interface{}) {
	if logger != nil {
		logWithLogger("FATAL", nil, format, values...)
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

type Signature struct {
	Worker       *SignatureWorker
	Service      *SignatureService
	JobName      string
	JobID        int64
	ProjectKey   string
	WorkflowName string
	WorkflowID   int64
	RunID        int64
	NodeRunName  string
	NodeRunID    int64
	Timestamp    int64
}

type SignatureWorker struct {
	WorkerID   string
	WorkerName string
	StepOrder  int64
	StepName   string
}

type SignatureService struct {
	HatcheryID      int64
	HatcheryName    string
	RequirementID   int64
	RequirementName string
	WorkerName      string
}

func New(ctx context.Context, graylogcfg *hook.Config) (*log.Logger, *hook.Hook, error) {
	newLogger := log.New()
	extra := map[string]interface{}{}
	hook, err := hook.NewHook(ctx, graylogcfg, extra)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to add hook: %v", err)
	}
	newLogger.AddHook(hook)
	return newLogger, hook, nil
}

func ReplaceAllHooks(ctx context.Context, l *log.Logger, graylogcfg *hook.Config) error {
	emptyHooks := log.LevelHooks{}
	oldHooks := l.ReplaceHooks(emptyHooks)
	for _, hooks := range oldHooks {
		for _, h := range hooks {
			varType := fmt.Sprintf("%T", h)

			if varType == fmt.Sprintf("%T", &hook.Hook{}) {
				log.Info("hatchery.ReplaceAllHooks> stopping previous hook")
				h.(*hook.Hook).Stop()
			}
		}
	}

	extra := map[string]interface{}{}
	hook, err := hook.NewHook(ctx, graylogcfg, extra)
	if err != nil {
		return fmt.Errorf("unable to add hook: %v", err)
	}
	l.AddHook(hook)
	return nil
}
