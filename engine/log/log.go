package log

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/spf13/viper"
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
)

// Logger defines the logs levels used
type Logger interface {
	Logf(fmt string, values ...interface{})
}

// Initialize initializes log level with flag --log-level
func Initialize() {
	switch viper.GetString("log_level") {
	case "debug":
		SetLevel(DebugLevel)
	case "info":
		SetLevel(InfoLevel)
	case "warning":
		SetLevel(WarningLevel)
	case "critical":
		SetLevel(CriticalLevel)
	case "notice":
		SetLevel(NoticeLevel)
	default:
		fmt.Fprintf(os.Stderr, "Invalid Log Level %s", viper.GetString("log_level"))
		os.Exit(1)
	}
}

// SetLogger override private logger reference
func SetLogger(l Logger) {
	logger = l
}

// SetLevel controls the categories of logs written
func SetLevel(lvl Level) {
	level = lvl
}

func lvl() Level {
	return level
}

// IsDebug returns true if current level is DebugLevel
func IsDebug() bool {
	return lvl() <= DebugLevel
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
	logger.Logf("[CRITICAL] "+format, values...)
}

// Fatalf prints fatal informations, then os.Exit(1)
func Fatalf(format string, values ...interface{}) {
	logger.Logf("[FATAL] "+format, values...)
	os.Exit(1)
}

// BaseLogger logs on stdout
type BaseLogger struct {
}

// Logf logs on stdout
func (l BaseLogger) Logf(fmt string, values ...interface{}) {
	log.Printf(fmt, values...)
}

// DatabaseLogger logs in database
type DatabaseLogger struct {
	db      *sql.DB
	logChan chan (dblog)
}

type dblog struct {
	logged time.Time
	level  string
	log    string
}

// Logf insert log into "system_log" table
func (l DatabaseLogger) Logf(format string, values ...interface{}) {
	go func() {
		line := fmt.Sprintf(format, values...)
		logged := time.Now()
		level := strings.Replace(strings.Trim(line[:10], "[] \n\t"), "\n", " ", -1)
		log := strings.Replace(strings.Trim(line[10:], " \n\t"), "\n", " ", -1)
		l.logChan <- dblog{logged: logged, level: level, log: log}
	}()
}

func (l DatabaseLogger) logger() {
	query := `INSERT INTO "system_log" (logged, level, log) VALUES ($1, $2, $3)`

	for {
		select {
		case log, ok := <-l.logChan:
			if !ok {
				return
			}
			_, err := l.db.Exec(query, log.logged, log.level, log.log)
			if err != nil {
				// Drop log since it's probably a db error anyway
				fmt.Printf("%s [%s] %s\n", log.logged, log.level, log.log)
			}
		}
	}
}

// UseDatabaseLogger should be used only with proper database
func UseDatabaseLogger(db *sql.DB) {
	l := DatabaseLogger{db: db, logChan: make(chan dblog)}
	go l.logger()
	logger = l
}

// RemovalRoutine removes logs older than 1 day from database
func RemovalRoutine(DBFunc func() *sql.DB) {
	for {
		time.Sleep(1 * time.Hour)
		db := DBFunc()
		if db != nil {
			query := `DELETE FROM system_log WHERE logged < NOW() - INTERVAL '1 days'`
			db.Exec(query)
		}
	}
}
