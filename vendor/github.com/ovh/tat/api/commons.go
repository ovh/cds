package main

import (
	"crypto/tls"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"runtime"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/gin-gonic/gin"
	loghook "github.com/ovh/logrus-ovh-hook"
	"github.com/ovh/tat"
	"github.com/spf13/viper"

	userDB "github.com/ovh/tat/api/user"
)

// PreCheckUser has to be called as a middleware on Gin Route.
// Check if username exists in database, return user if ok
func PreCheckUser(ctx *gin.Context) (tat.User, error) {
	var tatUser = tat.User{}
	found, err := userDB.FindByUsername(&tatUser, getCtxUsername(ctx))
	var e error
	if !found {
		e = errors.New("User unknown")
	} else if err != nil {
		e = errors.New("Error while fetching user")
	}
	if e != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": e})
		ctx.AbortWithError(http.StatusInternalServerError, e)
		return tatUser, e
	}
	return tatUser, nil
}

// GetParam returns the value of a parameter in Url.
// Example : http://host:port/:paramName
func GetParam(ctx *gin.Context, paramName string) (string, error) {
	value, found := ctx.Params.Get(paramName)
	if !found {
		s := paramName + " in url does not exist"
		ctx.JSON(http.StatusBadRequest, gin.H{"error": s})
		return "", errors.New(s)
	}
	return value, nil
}

// AbortWithReturnError abort gin context and return JSON to user with error details
func AbortWithReturnError(ctx *gin.Context, statusHTTP int, err error) {
	ctx.JSON(statusHTTP, gin.H{"error:": err.Error()})
	ctx.Abort()
}

// tatRecovery is a middleware that recovers from any panics and writes a 500 if there was one.
func tatRecovery(c *gin.Context) {
	defer func() {
		if err := recover(); err != nil {
			path := c.Request.URL.Path
			query := c.Request.URL.RawQuery
			username, _ := c.Get(tat.TatHeaderUsername)
			trace := make([]byte, 4096)
			count := runtime.Stack(trace, true)
			log.Panicf("[tatRecovery] err:%s method:%s path:%s query:%s username:%s stacktrace of %d bytes:%s",
				err, c.Request.Method, path, query, username, count, trace)

			c.AbortWithStatus(500)
		}
	}()
	c.Next()
}

var logFieldAppID string

func initLog() {
	if viper.GetBool("production") {
		// Only log the warning severity or above.
		log.SetLevel(log.InfoLevel)
		gin.SetMode(gin.ReleaseMode)
		log.SetFormatter(&log.JSONFormatter{})
	} else {
		log.SetLevel(log.DebugLevel)
	}

	logFieldAppID = viper.GetString("log_field_app_id")

	if viper.GetString("graylog_host") != "" && viper.GetString("graylog_port") != "" {
		graylogcfg := &loghook.Config{
			Addr:      fmt.Sprintf("%s:%s", viper.GetString("graylog_host"), viper.GetString("graylog_port")),
			Protocol:  viper.GetString("graylog_protocol"),
			TLSConfig: &tls.Config{ServerName: viper.GetString("graylog_host")},
		}

		var extra map[string]interface{}
		if viper.GetString("graylog_extra_key") != "" && viper.GetString("graylog_extra_value") != "" {
			extra = map[string]interface{}{
				viper.GetString("graylog_extra_key"): viper.GetString("graylog_extra_value"),
			}
		}

		h, err := loghook.NewHook(graylogcfg, extra)

		if err != nil {
			log.Errorf("Error while initialize graylog hook: %s", err)
		} else {
			log.AddHook(h)
			log.SetOutput(ioutil.Discard)
		}
	}
}

// ginrus returns a gin.HandlerFunc (middleware) that logs requests using logrus.
//
// Requests with errors are logged using logrus.Error().
// Requests without errors are logged using logrus.Info().
//
// It receives:
//   1. A time package format string (e.g. time.RFC3339).
//   2. A boolean stating whether to use UTC time zone or local.
func ginrus(l *log.Logger, timeFormat string, utc bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		// some evil middlewares modify this values
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery
		c.Next()

		end := time.Now()
		latency := end.Sub(start)
		if utc {
			end = end.UTC()
		}

		username, _ := c.Get(tat.TatHeaderUsername)
		tatReferer, _ := c.Get(tat.TatHeaderXTatRefererLower)
		sec := latency.Seconds()
		ms := int64(latency / time.Millisecond)

		entry := l.WithFields(log.Fields{
			"appID":                   logFieldAppID,
			"status":                  c.Writer.Status(),
			"method":                  c.Request.Method,
			"path":                    path,
			"query":                   query,
			"ip":                      c.ClientIP(),
			"latency":                 latency,
			"latency_nanosecond_int":  latency.Nanoseconds(),
			"latency_millisecond_int": ms,
			"latency_second_float":    sec,
			"user-agent":              c.Request.UserAgent(),
			"time":                    end.Format(timeFormat),
			"tatusername":             username,
			"tatfrom":                 tatReferer,
		})

		msg := fmt.Sprintf("%d %s %s %s %fs %dms %dns", c.Writer.Status(), c.Request.Method, path, username, sec, ms, latency)

		if len(c.Errors) > 0 {
			// Append error field if this is an erroneous request.
			entry.Error(fmt.Sprintf("ERROR %s %s", msg, c.Errors.String()))
		} else if c.Writer.Status() >= 400 {
			entry.Warn(fmt.Sprintf("WARN %s", msg))
		} else {
			entry.Info(fmt.Sprintf("INFO %s", msg))
		}
	}
}
