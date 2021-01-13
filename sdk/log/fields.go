package cdslog

import "github.com/rockbears/log"

const (
	// If you add a field constant, don't forget to add it in the log.RegisterField below
	AuthUserID     = log.Field("auth_user_id")
	AuthConsumerID = log.Field("auth_consumer_id")
	AuthSessionID  = log.Field("auth_session_id")
	Method         = log.Field("method")
	Route          = log.Field("route")
	RequestURI     = log.Field("request_uri")
	Deprecated     = log.Field("false")
	Handler        = log.Field("handler")
	Latency        = log.Field("latency")
	LatencyNum     = log.Field("latency_num")
	Status         = log.Field("status")
	StatusNum      = log.Field("status_num")
	Goroutine      = log.Field("goroutine")
	RequestID      = log.Field("request_id")
	Service        = log.Field("service")
	Stacktrace     = log.Field("stack_trace")
	Duration       = log.Field("duration_milliseconds_num")
	Size           = log.Field("size_num")
)

func init() {
	log.RegisterField(
		RequestID,
		Service,
		Stacktrace,
	)
}
