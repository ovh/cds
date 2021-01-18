package cdslog

import (
	"context"
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"testing"

	"github.com/rockbears/log"
	"github.com/sirupsen/logrus"

	"github.com/ovh/cds/sdk/cdn"
	"github.com/ovh/cds/sdk/log/hook"
)

// Conf contains log configuration
type Conf struct {
	Level                      string
	Format                     string
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
	HeaderRequestID      = "Request-ID"
	ExtraFieldSignature  = "Signature"
	ExtraFieldLine       = "Line"
	ExtraFieldTerminated = "Terminated"
)

var (
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

	switch conf.Format {
	case "discard":
		logrus.SetOutput(ioutil.Discard)
	case "json":
		logrus.SetFormatter(&logrus.JSONFormatter{})
	case "default":
		logrus.SetFormatter(&CDSFormatter{})
	}

	if conf.GraylogHost != "" && conf.GraylogPort != "" {
		if err := initGraylokHook(ctx, conf); err != nil {
			logrus.Error(err)
		}
	}
}

func initGraylokHook(ctx context.Context, conf *Conf) error {
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
			return fmt.Errorf("Error while initialize log: extraKey (len:%d) does not have same corresponding number of values on extraValue (len:%d)", len(keys), len(values))
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

	var err error
	graylogHook, err = hook.NewHook(ctx, graylogcfg, extra)
	if err != nil {
		return fmt.Errorf("unable to initialize graylog hook: %v", err)
	}
	logrus.AddHook(graylogHook)

	go func() {
		<-ctx.Done()
		log.Info(ctx, "Draining logs...")
		graylogHook.Flush()
	}()

	return nil
}

type Message struct {
	Value     string
	Level     logrus.Level
	Signature cdn.Signature
}

func (m Message) ServiceKey() string {
	return fmt.Sprintf("%d-%d", m.Signature.NodeRunID, m.Signature.Service.RequirementID)
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
