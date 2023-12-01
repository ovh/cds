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
	AuthHatcheryID     = log.Field("auth_hatchery_id")
	AuthUsername       = log.Field("auth_user_name")
	AuthWorkerName     = log.Field("auth_worker_name")
	RbackCheckerName   = log.Field("rbac_checker_name")
	Commit             = log.Field("commit")
	Deprecated         = log.Field("deprecated")
	Duration           = log.Field("duration_milliseconds_num")
	Goroutine          = log.Field("goroutine")
	Handler            = log.Field("handler")
	HookEventID        = log.Field("hook_event_id")
	IPAddress          = log.Field("ip_address")
	Latency            = log.Field("latency")
	LatencyNum         = log.Field("latency_num")
	Method             = log.Field("method")
	GpgKey             = log.Field("gpg_key")
	RbacRole           = log.Field("rbac_role")
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
	VCSServer          = log.Field("vcs_server")
	KafkaBroker        = log.Field("kafka_broker")
	KafkaTopic         = log.Field("kafka_topic")
	AnalyzeID          = log.Field("analyze_id")
	NodeRunID          = log.Field("node_run_id")
	PermJobID          = log.Field("permJobID")
	Workflow           = log.Field("workflow")
	WorkflowRunID      = log.Field("workflow_run_id")
	Component          = log.Field("component")
	Project            = log.Field("project")
)

func init() {
	log.RegisterField(
		Action,
		AuthUserID,
		AuthHatcheryID,
		AuthUsername,
		AuthServiceName,
		AuthServiceType,
		AuthWorkerName,
		AuthConsumerID,
		AuthSessionID,
		AuthSessionIAT,
		AuthSessionTokenID,
		Commit,
		Component,
		Deprecated,
		Duration,
		Goroutine,
		GpgKey,
		Handler,
		HookEventID,
		IPAddress,
		Latency,
		LatencyNum,
		Method,
		Project,
		RbackCheckerName,
		Repository,
		RbacRole,
		Route,
		RequestID,
		RequestURI,
		Service,
		Size,
		Stacktrace,
		Status,
		StatusNum,
		Sudo,
		VCSServer,
		KafkaBroker,
		KafkaTopic,
		AnalyzeID,
		NodeRunID,
		PermJobID,
		Workflow,
		WorkflowRunID,
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
