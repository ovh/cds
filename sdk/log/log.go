package log

import (
	"context"
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"

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

	ExtraFieldSignature  = "Signature"
	ExtraFieldLine       = "Line"
	ExtraFieldTerminated = "Terminated"
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

type Fields logrus.Fields

var _ Logger = new(TestingLogger)

func (t *TestingLogger) Logf(fmt string, values ...interface{}) {
	defer func() {
		if r := recover(); r != nil {
			logrus.StandardLogger().Logf(logrus.InfoLevel, fmt, values...)
		}
	}()
	t.t.Logf(fmt, values...)
}

func (t *TestingLogger) Errorf(fmt string, values ...interface{}) {
	defer func() {
		if r := recover(); r != nil {
			logrus.StandardLogger().Logf(logrus.ErrorLevel, fmt, values...)
		}
	}()
	t.t.Errorf(fmt, values...)
}

func (t *TestingLogger) Fatalf(fmt string, values ...interface{}) {
	defer func() {
		if r := recover(); r != nil {
			logrus.StandardLogger().Fatalf(fmt, values...)
		}
	}()
	t.t.Fatalf(fmt, values...)
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

func logWithLogger(level string, fields Fields, format string, values ...interface{}) {
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
		logrus.SetLevel(logrus.DebugLevel)
	case "info":
		logrus.SetLevel(logrus.InfoLevel)
	case "error":
		logrus.SetLevel(logrus.ErrorLevel)
	case "warning":
		logrus.SetLevel(logrus.WarnLevel)
	default:
		logrus.SetLevel(logrus.InfoLevel)
	}
	logrus.SetFormatter(&CDSFormatter{})

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
				logrus.Errorf("Error while initialize log: extraKey (len:%d) does not have same corresponding number of values on extraValue (len:%d)", len(keys), len(values))
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
			logrus.Errorf("Error while initialize graylog hook: %v", errhook)
		} else {
			logrus.AddHook(graylogHook)
			logrus.SetOutput(ioutil.Discard)
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
	logrus.Debugf(format, values...)
}

// InfoWithoutCtx prints information logrus.
func InfoWithoutCtx(format string, values ...interface{}) {
	Info(context.Background(), format, values...)
}

// Info prints information logrus.
func Info(ctx context.Context, format string, values ...interface{}) {
	InfoWithFields(ctx, nil, format, values...)
}

// InfoWithFields print info log with given logrus fields.
func InfoWithFields(ctx context.Context, fields Fields, format string, values ...interface{}) {
	if logger != nil {
		logWithLogger("INFO", fields, format, values...)
		return
	}
	newEntry(ctx, fields).Infof(format, values...)
}

// Warning prints warnings logrus.
func Warning(ctx context.Context, format string, values ...interface{}) {
	WarningWithFields(ctx, nil, format, values...)
}

// WarningWithFields print warning log with given logrus fields.
func WarningWithFields(ctx context.Context, fields Fields, format string, values ...interface{}) {
	if logger != nil {
		logWithLogger("WARN", fields, format, values...)
		return
	}
	newEntry(ctx, fields).Warningf(format, values...)
}

// Error prints error logrus.
func Error(ctx context.Context, format string, values ...interface{}) {
	ErrorWithFields(ctx, nil, format, values...)
}

// ErrorWithFields print error log with given logrus fields.
func ErrorWithFields(ctx context.Context, fields Fields, format string, values ...interface{}) {
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
	logrus.Fatalf(format, values...)
}

func newEntry(ctx context.Context, fields Fields) *logrus.Entry {
	entry := logrus.NewEntry(logrus.StandardLogger())
	if fields != nil {
		entry = entry.WithFields(logrus.Fields(fields))
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
		if f, ok := iFunc.(func(ctx context.Context) logrus.Fields); ok {
			contextFields := f(ctx)
			entry = entry.WithFields(contextFields)
		}
	}

	return entry
}

type Message struct {
	Value     string
	Level     logrus.Level
	Signature Signature
}

func (m Message) ServiceKey() string {
	return fmt.Sprintf("%d-%d", m.Signature.NodeRunID, m.Signature.Service.RequirementID)
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

func New(ctx context.Context, graylogcfg *hook.Config) (*logrus.Logger, *hook.Hook, error) {
	newLogger := logrus.New()
	extra := map[string]interface{}{}
	hook, err := hook.NewHook(ctx, graylogcfg, extra)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to add hook: %v", err)
	}
	newLogger.AddHook(hook)
	return newLogger, hook, nil
}

func ReplaceAllHooks(ctx context.Context, l *logrus.Logger, graylogcfg *hook.Config) error {
	emptyHooks := logrus.LevelHooks{}
	oldHooks := l.ReplaceHooks(emptyHooks)
	for _, hooks := range oldHooks {
		for _, h := range hooks {
			varType := fmt.Sprintf("%T", h)

			if varType == fmt.Sprintf("%T", &hook.Hook{}) {
				logrus.Info("hatchery.ReplaceAllHooks> stopping previous hook")
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
