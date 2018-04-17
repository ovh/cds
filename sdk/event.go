package sdk

import (
	"time"
)

const (
	EventSubsWorkflowRuns = "event:workflow:runs"
	EventSubWorkflowRun   = "event:workflow:run"
)

// Event represents a event from API
// Event is "create", "update", "delete"
// Status is  "Waiting" "Building" "Success" "Fail" "Unknown", optional
// DateEvent is a date (timestamp format)
type Event struct {
	Timestamp       time.Time              `json:"timestamp"`
	Hostname        string                 `json:"hostname"`
	CDSName         string                 `json:"cdsname"`
	EventType       string                 `json:"type_event"` // go type of payload
	Payload         map[string]interface{} `json:"payload"`
	Attempts        int                    `json:"attempt"`
	Username        string                 `json:"username,omitempty"`
	UserMail        string                 `json:"user_mail,omitempty"`
	ProjectKey      string                 `json:"project_key,omitempty"`
	ApplicationName string                 `json:"application_name,omitempty"`
	PipelineName    string                 `json:"pipeline_name,omitempty"`
	EnvironmentName string                 `json:"environment_name,omitempty"`
	WorkflowName    string                 `json:"workflow_name,omitempty"`
	WorkflowRunNum  int64                  `json:"workflow_run_num,omitempty"`
}

// EventSubscription data send to api to subscribe to an event
type EventSubscription struct {
	UUID         string `json:"uuid"`
	ProjectKey   string `json:"key"`
	WorkflowName string `json:"workflow_name"`
	WorkflowRuns bool   `json:"runs"`
	WorkflowNum  int64  `json:"num"`
	Overwrite    bool   `json:"overwrite"`
}

// EventEngine contains event data for engine
type EventEngine struct {
	Message string `json:"message"`
}

// EventWorkflowNodeJobRun contains event data for a workflow node run job
type EventRunWorkflowNodeJob struct {
	ID                int64  `json:"id"`
	WorkflowNodeRunID int64  `json:"workflow_node_run_id,omitempty"`
	Status            string `json:"status"`
	Queued            int64  `json:"queued,omitempty"`
	Start             int64  `json:"start,omitempty"`
	Done              int64  `json:"done,omitempty"`
	Model             string `json:"model,omitempty"`
}

// EventWorkflowNodeRun contains event data for a workflow node run
type EventRunWorkflowNode struct {
	ID                    int64                     `json:"id,omitempty"`
	NodeID                int64                     `json:"node_id,omitempty"`
	RunID                 int64                     `json:"run_id,omitempty"`
	Number                int64                     `json:"num,omitempty"`
	SubNumber             int64                     `json:"subnum,omitempty"`
	Status                string                    `json:"status,omitempty"`
	Start                 int64                     `json:"start,omitempty"`
	Done                  int64                     `json:"done,omitempty"`
	Payload               interface{}               `json:"payload,omitempty"`
	HookEvent             *WorkflowNodeRunHookEvent `json:"hook_event"`
	Manual                *WorkflowNodeRunManual    `json:"manual"`
	SourceNodeRuns        []int64                   `json:"source_node_runs"`
	WorkflowRunID         int64                     `json:"workflow_run_id"`
	RepositoryManagerName string                    `json:"repository_manager_name"`
	RepositoryFullName    string                    `json:"repository_full_name"`
	Hash                  string                    `json:"hash"`
	BranchName            string                    `json:"branch_name"`
	NodeName              string                    `json:"node_name"`
	StagesSummary         []StageSummary            `json:"stages_summary"`
}

// EventWorkflowRun contains event data for a workflow run
type EventRunWorkflow struct {
	ID            int64            `json:"id"`
	Number        int64            `json:"num"`
	Status        string           `json:"status"`
	Workflow      Workflow         `json:"workflow"`
	Start         int64            `json:"start"`
	LastExecution int64            `json:"last_execution"`
	LastModified  int64            `json:"last_modified"`
	Tags          []WorkflowRunTag `json:"tags"`
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
