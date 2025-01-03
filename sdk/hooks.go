package sdk

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/ovh/cds/sdk/glob"
	"github.com/rockbears/log"

	"database/sql/driver"
)

type HookListWorkflowRequest struct {
	HookEventUUID       string                `json:"hook_event_uuid"`
	VCSName             string                `json:"vcs_name"`
	RepositoryName      string                `json:"repository_name"`
	Ref                 string                `json:"ref"`
	PullRequestRefTo    string                `json:"pullrequest_ref_to"`
	Sha                 string                `json:"sha"`
	Paths               []string              `json:"paths"`
	RepositoryEventName WorkflowHookEventName `json:"repository_event"`
	RepositoryEventType WorkflowHookEventType `json:"repository_event_type"`
	AnalyzedProjectKeys StringSlice           `json:"project_keys"`
	Models              []EntityFullName      `json:"models"`
	Workflows           []EntityFullName      `json:"workflows"`
}

func IsValidHookPath(ctx context.Context, configuredPaths []string, paths []string) bool {
	if len(configuredPaths) == 0 {
		return true
	}
	if len(paths) == 0 {
		return false
	}
	regExps := make([]*regexp.Regexp, 0, len(configuredPaths))
	for _, p := range configuredPaths {
		regexpP, err := regexp.Compile(p)
		if err != nil {
			log.ErrorWithStackTrace(ctx, err)
			continue
		}
		regExps = append(regExps, regexpP)
	}

	for _, p := range paths {
		for _, r := range regExps {
			if r.MatchString(p) {
				return true
			}
		}
	}
	return false
}

func IsValidHookRefs(ctx context.Context, configuredRefs []string, currentEventRef string) bool {
	if len(configuredRefs) == 0 {
		return true
	}
	fullPattern := strings.Join(configuredRefs, " ")
	g := glob.New(fullPattern)
	result, err := g.MatchString(currentEventRef)
	if err != nil {
		log.ErrorWithStackTrace(ctx, err)
		return false
	}
	return result != nil
}

type HookAccessData struct {
	URL         string `json:"url" cli:"url"`
	HookSignKey string `json:"hook_sign_key" cli:"hook_sign_key"`
}

type Hook struct {
	UUID          string            `json:"uuid"`
	HookType      string            `json:"hook_type"`
	Configuration HookConfiguration `json:"configuration"`
	HookSignKey   string            `json:"hook_sign_key,omitempty"`
}
type HookConfiguration map[string]WorkflowNodeHookConfigValue

func (hc HookConfiguration) Value() (driver.Value, error) {
	j, err := json.Marshal(hc)
	return j, WrapError(err, "cannot marshal HookConfiguration")
}

func (hc *HookConfiguration) Scan(src interface{}) error {
	if src == nil {
		return nil
	}
	source, ok := src.([]byte)
	if !ok {
		return WithStack(fmt.Errorf("type assertion .([]byte) failed (%T)", src))
	}
	return WrapError(JSONUnmarshal(source, hc), "cannot unmarshal HookConfiguration")
}

// HookConfigValue represents the value of a node hook config
type HookConfigValue struct {
	Value              string   `json:"value"`
	Configurable       bool     `json:"configurable"`
	Type               string   `json:"type"`
	MultipleChoiceList []string `json:"multiple_choice_list"`
}

// Task is a generic hook tasks such as webhook, scheduler,... which will be started and wait for execution
type Task struct {
	UUID              string                 `json:"uuid" cli:"UUID,key"`
	Type              string                 `json:"type" cli:"Type"`
	Conditions        WorkflowNodeConditions `json:"conditions" cli:"Conditions"`
	Stopped           bool                   `json:"stopped" cli:"Stopped"`
	Executions        []TaskExecution        `json:"executions"`
	NbExecutionsTotal int                    `json:"nb_executions_total" cli:"nb_executions_total"`
	NbExecutionsTodo  int                    `json:"nb_executions_todo" cli:"nb_executions_todo"`
	Configuration     HookConfiguration      `json:"configuration" cli:"configuration"`
	// DEPRECATED
	Config WorkflowNodeHookConfig `json:"config" cli:"Config"`
}

// TaskExecution represents an execution instance of a task. It the task is a webhook; this represents the call of the webhook
type TaskExecution struct {
	UUID                string                  `json:"uuid" cli:"uuid,key"`
	Type                string                  `json:"type" cli:"type"`
	Timestamp           int64                   `json:"timestamp" cli:"timestamp"`
	NbErrors            int64                   `json:"nb_errors" cli:"nb_errors"`
	LastError           string                  `json:"last_error,omitempty" cli:"last_error"`
	ProcessingTimestamp int64                   `json:"processing_timestamp" cli:"processing_timestamp"`
	WorkflowRun         int64                   `json:"workflow_run" cli:"workflow_run"`
	WebHook             *WebHookExecution       `json:"webhook,omitempty" cli:"-"`
	Kafka               *KafkaTaskExecution     `json:"kafka,omitempty" cli:"-"`
	RabbitMQ            *RabbitMQTaskExecution  `json:"rabbitmq,omitempty" cli:"-"`
	ScheduledTask       *ScheduledTaskExecution `json:"scheduled_task,omitempty" cli:"-"`
	GerritEvent         *GerritEventExecution   `json:"gerrit,omitempty" cli:"-"`
	Status              string                  `json:"status" cli:"status"`
	Configuration       HookConfiguration       `json:"configuration" cli:"-"`
	// DEPRECATED
	Config WorkflowNodeHookConfig `json:"config" cli:"-"`
}

// GerritEventExecution contains specific data for a gerrit event execution
type GerritEventExecution struct {
	Message []byte `json:"message"`
}

// WebHookExecution contains specific data for a webhook execution
type WebHookExecution struct {
	RequestURL    string              `json:"request_url"`
	RequestBody   []byte              `json:"request_body"`
	RequestHeader map[string][]string `json:"request_header"`
	RequestMethod string              `json:"request_method"`
}

// KafkaTaskExecution contains specific data for a kafka hook
type KafkaTaskExecution struct {
	Message []byte `json:"message"`
}

// RabbitMQTaskExecution contains specific data for a kafka hook
type RabbitMQTaskExecution struct {
	Message []byte `json:"message"`
}

// ScheduledTaskExecution contains specific data for a scheduled task execution
type ScheduledTaskExecution struct {
	DateScheduledExecution string `json:"date_scheduled_execution"`
}
