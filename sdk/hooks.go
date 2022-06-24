package sdk

const (
	RepositoryEntitiesHook = "EntitiesHook"
	SignHeaderVCSName      = "X-Cds-Hooks-Vcs-Name"
	SignHeaderRepoName     = "X-Cds-Hooks-Repo-Name"
)

type RepositoryWebHook struct {
	UUID          string
	HookType      string
	Configuration HookConfiguration
}
type HookConfiguration map[string]WorkflowNodeHookConfigValue

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
