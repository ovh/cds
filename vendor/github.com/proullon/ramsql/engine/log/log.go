package log

import (
	// TODO: add a file logger
	"log"
	"sync"
	"testing"
)

func init() {
	level = WarningLevel
	logger = BaseLogger{}
}

// Level of logging trigger
type Level int

// Available logging levels
const (
	DebugLevel Level = iota
	InfoLevel
	NoticeLevel
	WarningLevel
	CriticalLevel
)

var (
	logger Logger
	level  Level
	mu     sync.Mutex
)

// Logger defines the logs levels used by RamSQL engine
type Logger interface {
	Logf(fmt string, values ...interface{})
}

// SetLevel controls the categories of logs written
func SetLevel(lvl Level) {
	mu.Lock()
	level = lvl
	mu.Unlock()
}

func lvl() Level {
	mu.Lock()
	defer mu.Unlock()
	return level
}

// Debug prints debug log
func Debug(format string, values ...interface{}) {
	if lvl() <= DebugLevel {
		logger.Logf("[DEBUG]    "+format, values...)
	}
}

// Info prints information log
func Info(format string, values ...interface{}) {
	if lvl() <= InfoLevel {
		logger.Logf("[INFO]     "+format, values...)
	}
}

// Notice prints information that should be seen
func Notice(format string, values ...interface{}) {
	if lvl() <= NoticeLevel {
		logger.Logf("[NOTICE]   "+format, values...)
	}
}

// Warning prints warnings for user
func Warning(format string, values ...interface{}) {
	if lvl() <= WarningLevel {
		logger.Logf("[WARNING]  "+format, values...)
	}
}

// Critical prints error informations
func Critical(format string, values ...interface{}) {
	mu.Lock()
	logger.Logf("[CRITICAL] "+format, values...)
	mu.Unlock()
}

// BaseLogger logs on stdout
type BaseLogger struct {
}

// Logf logs on stdout
func (l BaseLogger) Logf(fmt string, values ...interface{}) {
	log.Printf(fmt, values...)
}

// TestLogger uses *testing.T as a backend for RamSQL logs
type TestLogger struct {
	t *testing.T
}

// Logf logs in testing log buffer
func (l TestLogger) Logf(fmt string, values ...interface{}) {
	l.t.Logf(fmt, values...)
}

// UseTestLogger should be used only by unit tests
func UseTestLogger(t testing.TB) {
	mu.Lock()
	logger = t
	mu.Unlock()
	SetLevel(WarningLevel)
}
