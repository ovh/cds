package sdk

import (
	"encoding/json"
	"time"
)

// Event represents a event from API
// Event is "create", "update", "delete"
// Status is  "Waiting" "Building" "Success" "Fail" "Unknown", optional
// DateEvent is a date (timestamp format)
type Event struct {
	Timestamp           time.Time        `json:"timestamp"`
	Hostname            string           `json:"hostname"`
	CDSName             string           `json:"cdsname"`
	EventType           string           `json:"type_event"` // go type of payload
	Payload             json.RawMessage  `json:"payload"`
	Attempts            int              `json:"attempt"`
	Username            string           `json:"username,omitempty"`
	UserMail            string           `json:"user_mail,omitempty"`
	ProjectKey          string           `json:"project_key,omitempty"`
	ApplicationName     string           `json:"application_name,omitempty"`
	PipelineName        string           `json:"pipeline_name,omitempty"`
	EnvironmentName     string           `json:"environment_name,omitempty"`
	WorkflowName        string           `json:"workflow_name,omitempty"`
	WorkflowRunNum      int64            `json:"workflow_run_num,omitempty"`
	WorkflowRunNumSub   int64            `json:"workflow_run_num_sub,omitempty"`
	WorkflowNodeRunID   int64            `json:"workflow_node_run_id,omitempty"`
	OperationUUID       string           `json:"operation_uuid,omitempty"`
	Status              string           `json:"status,omitempty"`
	Tags                []WorkflowRunTag `json:"tag,omitempty"`
	EventIntegrationsID []int64          `json:"event_integrations_id"`
}

// EventFilter represents filters when getting events
type EventFilter struct {
	CurrentItem int            `json:"current_item"`
	Filter      TimelineFilter `json:"filter"`
}

// EventSubscription data send to api to subscribe to an event
type EventSubscription struct {
	UUID         string `json:"uuid"`
	ProjectKey   string `json:"key"`
	WorkflowName string `json:"workflow_name"`
	WorkflowNum  int64  `json:"num"`
	WorkflowRuns bool   `json:"runs"`
	Overwrite    bool   `json:"overwrite"`
}

// EventEngine contains event data for engine
type EventEngine struct {
	Message string `json:"message"`
}

// EventRunWorkflowNode contains event data for a workflow node run
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
	Tag                   string                    `json:"tag"`
	BranchName            string                    `json:"branch_name"`
	NodeName              string                    `json:"node_name"`
	StagesSummary         []StageSummary            `json:"stages_summary"`
	HookUUID              string                    `json:"hook_uuid"`
	HookLog               string                    `json:"log,omitempty"`
	NodeType              string                    `json:"node_type,omitempty"`
	GerritChange          *GerritChangeEvent        `json:"gerrit_change,omitempty"`
	EventIntegrations     []int64                   `json:"event_integrations_id,omitempty"`
}

// GerritChangeEvent Gerrit information that are needed on event
type GerritChangeEvent struct {
	ID         string `json:"id,omitempty"`
	Project    string `json:"project,omitempty"`
	DestBranch string `json:"dest_branch,omitempty"`
	Revision   string `json:"revision,omitempty"`
	Report     string `json:"report,omitempty"`
	URL        string `json:"url,omitempty"`
}

// EventRunWorkflowOutgoingHook contains event data for a workflow outgoing hook run
type EventRunWorkflowOutgoingHook struct {
	HookID            int64  `json:"hook_id"`
	ID                string `json:"id"`
	Status            string `json:"status,omitempty"`
	Start             int64  `json:"start,omitempty"`
	Done              int64  `json:"done,omitempty"`
	Log               string `json:"log,omitempty"`
	WorkflowRunID     int64  `json:"workflow_run_id"`
	WorkflowRunNumber *int64 `json:"workflow_run_number,omitempty"`
}

// EventRunWorkflowJob contains event data for a workflow job node run
type EventRunWorkflowJob struct {
	ID     int64  `json:"id,omitempty"`
	Status string `json:"status,omitempty"`
	Start  int64  `json:"start,omitempty"`
	Done   int64  `json:"done,omitempty"`
}

// EventRunWorkflow contains event data for a workflow run
type EventRunWorkflow struct {
	ID               int64            `json:"id"`
	Number           int64            `json:"num"`
	Status           string           `json:"status"`
	Start            int64            `json:"start"`
	LastExecution    int64            `json:"last_execution"`
	LastModified     int64            `json:"last_modified"`
	LastModifiedNano int64            `json:"last_modified_nano"`
	Tags             []WorkflowRunTag `json:"tags"`
	ToDelete         bool             `json:"to_delete"`
}

// EventNotif contains event data for a job
type EventNotif struct {
	Recipients []string `json:"recipients"`
	Subject    string   `json:"subject,omitempty"`
	Body       string   `json:"body,omitempty"`
}

// EventMaintenance contains event data for maintenance event
type EventMaintenance struct {
	Enable bool `json:"enable"`
}

// EventFake is used for test purpose
type EventFake struct {
	Data int64 `json:"data"`
}
