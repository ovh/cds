package cdslog

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"log/syslog"
	"os"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/rockbears/log"
	"github.com/sirupsen/logrus"
	lSyslog "github.com/sirupsen/logrus/hooks/syslog"

	"github.com/ovh/cds/sdk/cdn"
	"github.com/ovh/cds/sdk/log/hook"
)

// Conf contains log configuration
type Conf struct {
	Level                      string
	Format                     string
	TextFields                 []string
	SkipTextFields             []string
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
	SyslogHost                 string
	SyslogPort                 string
	SyslogProtocol             string
	SyslogExtraTag             string
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
		logrus.SetOutput(io.Discard)
	case "json":
		logrus.SetFormatter(&logrus.JSONFormatter{})
	default:
		for _, v := range conf.SkipTextFields {
			t := strings.SplitN(v, "=", 2)
			fieldName := t[0]
			fieldValue := t[1]
			log.Skip(log.Field(fieldName), fieldValue)
		}
		logrus.SetFormatter(&CDSFormatter{Fields: conf.TextFields})
	}

	if conf.GraylogHost != "" && conf.GraylogPort != "" {
		if err := initGraylogHook(ctx, conf); err != nil {
			logrus.Error(err)
		}
	}

	if conf.SyslogHost != "" && conf.SyslogPort != "" {
		if err := initSyslogHook(conf); err != nil {
			logrus.Error(err)
		}
	}
}

func initSyslogHook(conf *Conf) (err error) {
	logrus.Infoln("initializing Syslog hook...")
	defer func() {
		time.Sleep(time.Second)
		if err != nil {
			log.Error(context.Background(), "unable to initialize syslog hook on %s: %v", conf.SyslogHost+":"+conf.SyslogPort, err)
		} else {
			log.Info(context.Background(), "syslog hook initialized")
		}
	}()

	hook, err := lSyslog.NewSyslogHook(conf.SyslogProtocol, conf.SyslogHost+":"+conf.SyslogPort, syslog.LOG_INFO, conf.SyslogExtraTag)
	if err != nil {
		return errors.Wrap(err, "unable to init syslog hook")
	}

	logrus.AddHook(hook)

	return nil
}

func initGraylogHook(ctx context.Context, conf *Conf) (err error) {
	logrus.Infoln("initializing Graylog hook...")

	defer func() {
		time.Sleep(time.Second)
		if err != nil {
			log.Error(context.Background(), "unable to initialize graylog hook on %s: %v", conf.GraylogHost+":"+conf.GraylogPort, err)
		} else {
			log.Info(context.Background(), "graylog hook initialized")
		}
	}()

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
			return errors.Errorf("error while initialize log: extraKey (len:%d) does not have same corresponding number of values on extraValue (len:%d)", len(keys), len(values))
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
	if m.Signature.Service != nil {
		return fmt.Sprintf("%d-%d", m.Signature.NodeRunID, m.Signature.Service.RequirementID)
	}
	return fmt.Sprintf("%s-%s", m.Signature.RunJobID, m.Signature.HatcheryService.ServiceName)
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

// For given logrus logger, try to flush hooks
func Flush(ctx context.Context, l *logrus.Logger) {
	for _, hs := range logrus.StandardLogger().Hooks {
		for _, h := range hs {
			if graylogHook, ok := h.(*hook.Hook); ok {
				log.Info(ctx, "Draining logs...")
				graylogHook.Flush()
			}
		}
	}
}
