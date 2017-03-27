package sdk

import (
	"time"
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
	Attempts  int                    `json:"attempt"`
}

// EventEngine contains event data for engine
type EventEngine struct {
	Message string `json:"message"`
}

// EventPipelineBuild contains event data for a pipeline build
type EventPipelineBuild struct {
	Version               int64  `json:"version,omitempty"`
	BuildNumber           int64  `json:"buildNumber,omitempty"`
	Status                Status `json:"status,omitempty"`
	Start                 int64  `json:"start,omitempty"`
	Done                  int64  `json:"done,omitempty"`
	PipelineName          string `json:"pipelineName,omitempty"`
	PipelineType          string `json:"type,omitempty"`
	ProjectKey            string `json:"projectKey,omitempty"`
	ApplicationName       string `json:"applicationName,omitempty"`
	EnvironmentName       string `json:"environmentName,omitempty"`
	BranchName            string `json:"branchName,omitempty"`
	Hash                  string `json:"hash,omitempty"`
	RepositoryManagerName string `json:"repositoryManagerName,omitempty"`
	RepositoryFullname    string `json:"repositoryFullname,omitempty"`
}

// EventJob contains event data for a job
type EventJob struct {
	Version         int64  `json:"version,omitempty"`
	JobName         string `json:"jobName,omitempty"`
	JobID           int64  `json:"jobID,omitempty"`
	Status          Status `json:"status,omitempty"`
	Queued          int64  `json:"queued,omitempty"`
	Start           int64  `json:"start,omitempty"`
	Done            int64  `json:"done,omitempty"`
	ModelName       string `json:"modelName,omitempty"`
	PipelineName    string `json:"pipelineName,omitempty"`
	PipelineType    string `json:"type,omitempty"`
	ProjectKey      string `json:"projectKey,omitempty"`
	ApplicationName string `json:"applicationName,omitempty"`
	EnvironmentName string `json:"environmentName,omitempty"`
	BranchName      string `json:"branchName,omitempty"`
	Hash            string `json:"hash,omitempty"`
}

// EventNotif contains event data for a job
type EventNotif struct {
	Recipients []string `json:"recipients"`
	Subject    string   `json:"subject,omitempty"`
	Body       string   `json:"body,omitempty"`
}
