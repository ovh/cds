package cdslog

import (
	"context"

	"github.com/rockbears/log"
)

const (
	// If you add a field constant, don't forget to add it in the log.RegisterField below
	Action             = log.Field("action")
	AuthConsumerID     = log.Field("auth_consumer_id")
	AuthServiceName    = log.Field("auth_service_name")
	AuthServiceType    = log.Field("auth_service_type")
	AuthSessionIAT     = log.Field("auth_session_iat")
	AuthSessionID      = log.Field("auth_session_id")
	AuthSessionTokenID = log.Field("auth_session_token")
	AuthUserID         = log.Field("auth_user_id")
	AuthUsername       = log.Field("auth_user_name")
	AuthWorkerName     = log.Field("auth_worker_name")
	RbackCheckerName   = log.Field("rbac_checker_name")
	Deprecated         = log.Field("deprecated")
	Duration           = log.Field("duration_milliseconds_num")
	Goroutine          = log.Field("goroutine")
	Handler            = log.Field("handler")
	IPAddress          = log.Field("ip_address")
	Latency            = log.Field("latency")
	LatencyNum         = log.Field("latency_num")
	Method             = log.Field("method")
	RequestID          = log.Field("request_id")
	RequestURI         = log.Field("request_uri")
	Repository         = log.Field("repository")
	Operation          = log.Field("operation")
	Route              = log.Field("route")
	Service            = log.Field("service")
	Size               = log.Field("size_num")
	Stacktrace         = log.Field("stack_trace")
	Status             = log.Field("status")
	StatusNum          = log.Field("status_num")
	Sudo               = log.Field("sudo")
)

func init() {
	log.RegisterField(
		Action,
		AuthUserID,
		AuthUsername,
		AuthServiceName,
		AuthServiceType,
		AuthWorkerName,
		AuthConsumerID,
		AuthSessionID,
		AuthSessionIAT,
		AuthSessionTokenID,
		Deprecated,
		Duration,
		Goroutine,
		Handler,
		IPAddress,
		Latency,
		LatencyNum,
		Method,
		RbackCheckerName,
		Route,
		RequestID,
		RequestURI,
		Service,
		Size,
		Stacktrace,
		Status,
		StatusNum,
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
