package sdk

// Task is a generic hook tasks such as webhook, scheduler,... which will be started and wait for execution
type Task struct {
	UUID       string                 `json:"uuid" cli:"UUID,key"`
	Type       string                 `json:"type" cli:"Type"`
	Config     WorkflowNodeHookConfig `json:"config" cli:"Config"`
	Stopped    bool                   `json:"stopped" cli:"Stopped"`
	Executions []TaskExecution        `json:"executions"`
}

// TaskExecution represents an execution instance of a task. It the task is a webhook; this represents the call of the webhook
type TaskExecution struct {
	UUID                string                  `json:"uuid"`
	Type                string                  `json:"type"`
	Timestamp           int64                   `json:"timestamp"`
	NbErrors            int64                   `json:"nb_errors"`
	LastError           string                  `json:"last_error,omitempty"`
	ProcessingTimestamp int64                   `json:"processing_timestamp"`
	WorkflowRun         int64                   `json:"workflow_run"`
	Config              WorkflowNodeHookConfig  `json:"config"`
	WebHook             *WebHookExecution       `json:"webhook,omitempty"`
	ScheduledTask       *ScheduledTaskExecution `json:"scheduled_task,omitempty"`
	Status              string                  `json:"status"`
}

// WebHookExecution contains specific data for a webhook execution
type WebHookExecution struct {
	RequestURL    string              `json:"request_url"`
	RequestBody   []byte              `json:"request_body"`
	RequestHeader map[string][]string `json:"request_header"`
}

// KafkaTestExecution contains specific data for a kafka hook
type KafkaTaskExecution struct {
	Message []byte
}

// ScheduledTaskExecution contains specific data for a scheduled task execution
type ScheduledTaskExecution struct {
	DateScheduledExecution string `json:"date_scheduled_execution"`
}
