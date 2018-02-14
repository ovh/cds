package sdk

// Task is a generic hook tasks such as webhook, scheduler,... which will be started and wait for execution
type Task struct {
	UUID       string                 `cli:"UUID,key"`
	Type       string                 `cli:"Type"`
	Config     WorkflowNodeHookConfig `cli:"Config"`
	Stopped    bool                   `cli:"Stopped"`
	Executions []TaskExecution
}

// TaskExecution represents an execution instance of a task. It the task is a webhook; this represents the call of the webhook
type TaskExecution struct {
	UUID                string
	Type                string
	Timestamp           int64
	NbErrors            int64
	LastError           string
	ProcessingTimestamp int64
	WorkflowRun         int64
	Config              WorkflowNodeHookConfig
	WebHook             *WebHookExecution
	ScheduledTask       *ScheduledTaskExecution
	Status              string
}

// WebHookExecution contains specific data for a webhook execution
type WebHookExecution struct {
	RequestURL    string
	RequestBody   []byte
	RequestHeader map[string][]string
}

// ScheduledTaskExecution contains specific data for a scheduled task execution
type ScheduledTaskExecution struct {
	DateScheduledExecution string
}
