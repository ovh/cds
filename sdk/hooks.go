package sdk

// Task is a generic hook tasks such as webhook, scheduler,... which will be started and wait for execution
type Task struct {
	UUID              string                 `json:"uuid" cli:"UUID,key"`
	Type              string                 `json:"type" cli:"Type"`
	Config            WorkflowNodeHookConfig `json:"config" cli:"Config"`
	Stopped           bool                   `json:"stopped" cli:"Stopped"`
	Executions        []TaskExecution        `json:"executions"`
	NbExecutionsTotal int                    `json:"nb_executions_total" cli:"nb_executions_total"`
	NbExecutionsTodo  int                    `json:"nb_executions_todo" cli:"nb_executions_todo"`
}

// TaskExecution represents an execution instance of a task. It the task is a webhook; this represents the call of the webhook
type TaskExecution struct {
	UUID                string                  `json:"uuid" cli:"uuid"`
	Type                string                  `json:"type" cli:"type"`
	Timestamp           int64                   `json:"timestamp" cli:"timestamp"`
	NbErrors            int64                   `json:"nb_errors" cli:"nb_errors"`
	LastError           string                  `json:"last_error,omitempty" cli:"last_error"`
	ProcessingTimestamp int64                   `json:"processing_timestamp" cli:"processing_timestamp"`
	WorkflowRun         int64                   `json:"workflow_run" cli:"workflow_run"`
	Config              WorkflowNodeHookConfig  `json:"config" cli:"-"`
	WebHook             *WebHookExecution       `json:"webhook,omitempty" cli:"-"`
	Kafka               *KafkaTaskExecution     `json:"kafka,omitempty" cli:"-"`
	RabbitMQ            *RabbitMQTaskExecution  `json:"rabbitmq,omitempty" cli:"-"`
	ScheduledTask       *ScheduledTaskExecution `json:"scheduled_task,omitempty" cli:"-"`
	Status              string                  `json:"status" cli:"status"`
}

// WebHookExecution contains specific data for a webhook execution
type WebHookExecution struct {
	RequestURL    string              `json:"request_url"`
	RequestBody   []byte              `json:"request_body"`
	RequestHeader map[string][]string `json:"request_header"`
}

// KafkaTaskExecution contains specific data for a kafka hook
type KafkaTaskExecution struct {
	Message []byte
}

// RabbitMQTaskExecution contains specific data for a kafka hook
type RabbitMQTaskExecution struct {
	Message []byte
}

// ScheduledTaskExecution contains specific data for a scheduled task execution
type ScheduledTaskExecution struct {
	DateScheduledExecution string `json:"date_scheduled_execution"`
}
