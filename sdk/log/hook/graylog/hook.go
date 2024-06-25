// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.
// inspired from github.com/gemnasium/logrus-graylog-hook

package graylog

import (
	"context"
	"crypto/tls"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/eapache/go-resiliency/retrier"
	"github.com/sirupsen/logrus"
)

type Priority int

const (
	// Severity.

	// From /usr/include/sys/syslog.h.
	// These are the same on Linux, BSD, and OS X.
	LOG_EMERG Priority = iota
	LOG_ALERT
	LOG_CRIT
	LOG_ERR
	LOG_WARNING
	LOG_NOTICE
	LOG_INFO
	LOG_DEBUG
)

// priorities maps logrus log levels to syslog severity
var priorities = map[logrus.Level]Priority{
	logrus.PanicLevel: LOG_ALERT,
	logrus.FatalLevel: LOG_CRIT,
	logrus.ErrorLevel: LOG_ERR,
	logrus.WarnLevel:  LOG_WARNING,
	logrus.InfoLevel:  LOG_INFO,
	logrus.DebugLevel: LOG_DEBUG,
}

// MergeFields defines a function to merge fields. It used for example to define your own field
// convientions to match with your graylog service.
type MergeFields func(...map[string]interface{}) map[string]interface{}

// Set graylog.BufSize = <value> _before_ calling NewHook
// Once the buffer is full, logging will start blocking, waiting for slots to
// be available in the queue.
var BufSize uint = 16384

// Config is the required configuration for creating a Graylog hook
type Config struct {
	Addr           string
	Protocol       string
	Hostname       string
	Facility       string
	TLSConfig      *tls.Config
	Merge          func(...map[string]interface{}) map[string]interface{}
	ThrottlePolicy *ThrottlePolicyConfig
}

// Hook to send logs to a logging service compatible with the Graylog API and the GELF format.
type Hook struct {
	Facility string
	Hostname string
	// Extra fields to send to Graylog for each log entry.
	Extra map[string]interface{}
	// Minimum logging level to send to Graylog.
	// Must be set before adding to logrus logger.
	// Default is logrus.InfoLevel.
	Threshold logrus.Level

	merge      MergeFields
	Pid        int
	gelfLogger Writer

	throttleStack  *Stack
	throttleTicker *time.Ticker
	throttlePolicy ThrottlePolicy

	stopChan chan bool
}

// NewHook creates a hook to be added to an instance of logger.
func NewHook(ctx context.Context, cfg *Config, extra map[string]interface{}) (*Hook, error) {
	// Get a hostname if not set
	hostname := cfg.Hostname
	if hostname == "" {
		if h, err := os.Hostname(); err == nil {
			if i := strings.Index(h, "."); i >= 0 {
				h = h[:i]
			}
			hostname = h
		}
	}

	// Get protocol
	protocol := cfg.Protocol
	if protocol == "" {
		protocol = "tcp"
	}

	// Join host and port
	var w Writer
	var err error

	switch protocol {
	case "tcp":
		w, err = NewTCPWriter(cfg.Addr, cfg.TLSConfig)
	case "udp":
		w, err = NewUDPWriter(cfg.Addr)
	default:
		err = fmt.Errorf("unknown protocol %q", protocol)
	}

	if err != nil {
		return nil, err
	}

	merge := mergeFields
	if cfg.Merge != nil {
		merge = cfg.Merge
	}

	hook := &Hook{
		Facility:   cfg.Facility,
		Hostname:   hostname,
		Extra:      extra,
		Threshold:  logrus.DebugLevel,
		merge:      merge,
		Pid:        os.Getpid(),
		gelfLogger: w,
		stopChan:   make(chan bool),
	}

	if cfg.ThrottlePolicy == nil {
		dfault := NewDefaultThrottlePolicy()
		cfg.ThrottlePolicy = &ThrottlePolicyConfig{
			Amount: 5,
			Period: 10 * time.Second,
			Policy: dfault,
		}
	}

	hook.throttleStack = NewStack(cfg.ThrottlePolicy.Amount)
	hook.throttleTicker = time.NewTicker(cfg.ThrottlePolicy.Period)
	hook.throttlePolicy = cfg.ThrottlePolicy.Policy
	hook.throttlePolicy.Init(hook)
	fmt.Fprintf(os.Stdout, "[graylog] using endpoint: %s\n", cfg.Addr)

	go hook.fire() // Log in background

	go func() {
		select {
		case <-ctx.Done():
			hook.Flush()
		case <-hook.stopChan:
			return
		}

	}()

	return hook, nil
}

func (hook *Hook) Stop() {
	if hook.stopChan != nil {
		hook.stopChan <- true
		close(hook.stopChan)
		hook.stopChan = nil
	}
}

// Fire is called when a log event is fired.
// We assume the entry will be altered by another hook,
// otherwise we might logging something wrong to Graylog
func (hook *Hook) Fire(entry *logrus.Entry) error {
	// get caller file and line here, it won't be available inside the goroutine
	// 1 for the function that called us.
	// we also make most of the work out of the lock scope to reduce
	// performance impact due to locking
	file, line := getCallerIgnoringLogMulti(1)
	msg := hook.messageFromEntry(entry, file, line)
	return hook.FireMessage(*msg)
}

type FireMessageFunc func(msg Message) error

func (hook *Hook) IsThrottled() bool {
	return !hook.throttleStack.Empty()
}

func (hook *Hook) FireMessage(msg Message) error {
	switch {
	case !hook.IsThrottled() && !hook.throttlePolicy.PendingTrailingMessages():
		hook.throttleStack.Push(msg)
	default:
		hook.throttlePolicy.HandleTrailingMessage(msg)
	}
	return nil
}

func (hook *Hook) Flush() {
	hook.throttlePolicy.Flush()
}

var r = retrier.New(retrier.ExponentialBackoff(20, time.Millisecond), nil)

func (hook *Hook) send(m Message) {
	// we retry at least 3 times to write message to graylog.
	err := r.Run(func() error {
		if err := hook.gelfLogger.WriteMessage(&m); err != nil {
			fmt.Fprintln(os.Stderr, "[graylog] could not write message to Graylog:", err)
			return err
		}
		return nil
	})
	// if after all the retries we still cannot write the message, just skip
	if err != nil {
		fmt.Fprintln(os.Stderr, "[graylog] could not write message to Graylog after several retries:", err)
	}
}

// fire will loop on the 'throttled' channel, and write entries to graylog
func (hook *Hook) fire() {
	defer hook.throttleTicker.Stop()
	for range hook.throttleTicker.C {
		for {
			time.Sleep(time.Millisecond)
			m, has := hook.throttleStack.Pop()
			if !has {
				continue
			}
			hook.send(m)
		}
	}
}

// Levels returns the available logging levels.
func (hook *Hook) Levels() []logrus.Level {
	levels := make([]logrus.Level, 0, hook.Threshold)

	for l := logrus.PanicLevel; l <= hook.Threshold; l++ {
		levels = append(levels, l)
	}

	return levels
}

func (hook *Hook) messageFromEntry(entry *logrus.Entry, file string, line int) *Message {
	p := entry.Message

	// If there are newlines in the message, use the first line
	// for the short message and set the full message to the
	// original input.  If the input has no newlines, stick the
	// whole thing in Short.
	short := p
	full := p
	if i := strings.IndexRune(p, '\n'); i > 0 {
		short = p[:i]
		full = p
	}

	// Merge hook extra fields and entry fields
	extra := hook.merge(hook.Extra, entry.Data)
	return &Message{
		Version:  "1.1",
		Host:     hook.Hostname,
		Short:    short,
		Full:     full,
		Time:     float64(entry.Time.UnixNano()) / 1e9,
		Level:    int32(priorities[entry.Level]),
		Pid:      hook.Pid,
		Facility: hook.Facility,
		File:     file,
		Line:     line,
		Extra:    extra,
	}
}

func mergeFields(extraFields ...map[string]interface{}) map[string]interface{} {
	mergedFields := make(map[string]interface{})
	for _, fields := range extraFields {
		for fieldName, value := range fields {
			// skip id if present
			if fieldName == "id" {
				continue
			}

			// otherwise convert if necessary
			switch value.(type) {
			// if string or number
			case string, int, int8, int16, int32, int64,
				uint, uint8, uint16, uint32, uint64,
				float32, float64:
				mergedFields["_"+fieldName] = value
			case time.Time:
				mergedFields["_"+fieldName] = value.(time.Time).Format(time.RFC3339)
			default:
				mergedFields["_"+fieldName] = fmt.Sprintf("%v", value)
			}
		}
	}

	return mergedFields
}
