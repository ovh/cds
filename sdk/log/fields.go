package cdslog

import (
	"context"

	"github.com/rockbears/log"
)

const (
	// If you add a field constant, don't forget to add it in the log.RegisterField below
	AuthUserID         = log.Field("auth_user_id")
	AuthUsername       = log.Field("auth_user_name")
	AuthServiceName    = log.Field("auth_service_name")
	AuthWorkerName     = log.Field("auth_worker_name")
	AuthConsumerID     = log.Field("auth_consumer_id")
	AuthSessionID      = log.Field("auth_session_id")
	AuthSessionIAT     = log.Field("auth_session_iat")
	AuthSessionTokenID = log.Field("auth_session_token")
	Method             = log.Field("method")
	Route              = log.Field("route")
	RequestURI         = log.Field("request_uri")
	Deprecated         = log.Field("deprecated")
	Handler            = log.Field("handler")
	Action             = log.Field("action")
	Latency            = log.Field("latency")
	LatencyNum         = log.Field("latency_num")
	Status             = log.Field("status")
	StatusNum          = log.Field("status_num")
	Goroutine          = log.Field("goroutine")
	RequestID          = log.Field("request_id")
	Service            = log.Field("service")
	Stacktrace         = log.Field("stack_trace")
	Duration           = log.Field("duration_milliseconds_num")
	Size               = log.Field("size_num")
	Sudo               = log.Field("sudo")
)

func init() {
	log.RegisterField(
		Action,
		AuthUserID,
		AuthUsername,
		AuthServiceName,
		AuthWorkerName,
		AuthConsumerID,
		AuthSessionID,
		AuthSessionIAT,
		AuthSessionTokenID,
		Method,
		Route,
		RequestURI,
		Deprecated,
		Handler,
		Latency,
		LatencyNum,
		Status,
		StatusNum,
		Goroutine,
		RequestID,
		Service,
		Stacktrace,
		Duration,
		Size,
		Sudo,
	)
}

func ContextValue(ctx context.Context, f log.Field) string {
	i := ctx.Value(f)
	if i != nil {
		if s, ok := i.(string); ok {
			return s
		}
	}
	return ""
}
