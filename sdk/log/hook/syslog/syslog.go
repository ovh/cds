package syslog

import (
	"crypto/tls"
	"fmt"
	"os"

	syslog "github.com/RackSec/srslog"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type Config struct {
	Protocol  string
	Address   string
	TLSConfig *tls.Config
	Tag       string
}

func NewHook(config Config) (*SyslogHook, error) {
	if config.TLSConfig != nil {
		w, err := syslog.DialWithTLSConfig("tcp+tls", config.Address, syslog.LOG_INFO, config.Tag, config.TLSConfig)
		if err != nil {
			return nil, errors.WithStack(err)
		}
		return &SyslogHook{Writer: w}, nil
	}

	w, err := syslog.Dial(config.Protocol, config.Address, syslog.LOG_INFO, config.Tag)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return &SyslogHook{Writer: w}, nil
}

type SyslogHook struct {
	Writer *syslog.Writer
}

func (hook *SyslogHook) Fire(entry *logrus.Entry) error {
	line, err := entry.String()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to read entry, %v", err)
		return err
	}

	switch entry.Level {
	case logrus.PanicLevel:
		return hook.Writer.Crit(line)
	case logrus.FatalLevel:
		return hook.Writer.Crit(line)
	case logrus.ErrorLevel:
		return hook.Writer.Err(line)
	case logrus.WarnLevel:
		return hook.Writer.Warning(line)
	case logrus.InfoLevel:
		return hook.Writer.Info(line)
	case logrus.DebugLevel, logrus.TraceLevel:
		return hook.Writer.Debug(line)
	default:
		return nil
	}
}

func (hook *SyslogHook) Levels() []logrus.Level {
	return logrus.AllLevels
}
