package sdk

import (
	"time"
)

// EventSource reprensents a type of event
type EventSource string

// Type of event
const (
	// UserEvent represent an event that a user want to have
	// a mail, a jabb
	UserEvent   EventSource = "userEvent"
	SystemEvent EventSource = "systemEvent"
)

// Event represents a event from API
// Event is "create", "update", "delete"
// Status is  "Waiting" "Building" "Success" "Fail" "Unknown", optional
// DateEvent is a date (timestamp format)
type Event struct {
	Timestamp time.Time              `json:"timestamp"`
	Hostname  string                 `json:"hostname"`
	CDSName   string                 `json:"cdsname"`
	EventType string                 `json:"type_event"` // go type of payload
	Payload   map[string]interface{} `json:"payload"`
}

// EventEngine contains event data for engine
type EventEngine struct {
	Message string `json:"message"`
}

// EventPipelineBuild contains event data for a pipeline build
type EventPipelineBuild struct {
	Version         int64        `json:"version,omitempty"`
	Status          Status       `json:"status,omitempty"`
	Start           time.Time    `json:"start,omitempty"`
	Done            time.Time    `json:"done,omitempty"`
	PipelineName    string       `json:"pipelineName,omitempty"`
	PipelineType    PipelineType `json:"type,omitempty"`
	ProjectKey      string       `json:"projectKey,omitempty"`
	ApplicationName string       `json:"applicationName,omitempty"`
	EnvironmentName string       `json:"environmentName,omitempty"`
	BranchName      string       `json:"branchName,omitempty"`
}

// EventJob contains event data for a job
type EventJob struct {
	Version         int64        `json:"version,omitempty"`
	JobName         string       `json:"jobName,omitempty"`
	Status          Status       `json:"status,omitempty"`
	Queued          time.Time    `json:"queued,omitempty"`
	Start           time.Time    `json:"start,omitempty"`
	Done            time.Time    `json:"done,omitempty"`
	Model           string       `json:"model,omitempty"`
	PipelineName    string       `json:"pipelineName,omitempty"`
	PipelineType    PipelineType `json:"type,omitempty"`
	ProjectKey      string       `json:"projectKey,omitempty"`
	ApplicationName string       `json:"applicationName,omitempty"`
	EnvironmentName string       `json:"environmentName,omitempty"`
	BranchName      string       `json:"branchName,omitempty"`
}

// EventNotif contains event data for a job
type EventNotif struct {
	Recipients []string `json:"recipients"`
	Subject    string   `json:"subject,omitempty"`
	Body       string   `json:"body,omitempty"`
}
