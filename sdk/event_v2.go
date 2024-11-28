package sdk

import (
	"encoding/json"
	"time"
)

type EventType string

const (
	EventAnalysisStart EventType = "AnalysisStart"
	EventAnalysisDone  EventType = "AnalysisDone"

	EventRunJobEnqueued         EventType = "RunJobEnqueued"
	EventRunJobScheduled        EventType = "RunJobScheduled"
	EventRunJobBuilding         EventType = "RunJobBuilding"
	EventRunJobManualTriggered  EventType = "RunJobManualTriggered"
	EventRunJobRunResultAdded   EventType = "RunJobRunResultAdded"
	EventRunJobRunResultUpdated EventType = "RunJobRunResultUpdated"
	EventRunJobEnded            EventType = "RunJobEnded"

	EventRunCrafted  EventType = "RunCrafted"
	EventRunBuilding EventType = "RunBuilding"
	EventRunEnded    EventType = "RunEnded"
	EventRunRestart  EventType = "RunRestart"

	EventEntityCreated EventType = "EntityCreated"
	EventEntityUpdated EventType = "EntityUpdated"
	EventEntityDeleted EventType = "EntityDeleted"

	EventVCSCreated EventType = "VCSCreated"
	EventVCSUpdated EventType = "VCSUpdated"
	EventVCSDeleted EventType = "VCSDeleted"

	EventHatcheryCreated    EventType = "HatcheryCreated"
	EventHatcheryUpdated    EventType = "HatcheryUpdated"
	EventHatcheryTokenRegen EventType = "HatcheryTokenRegen"
	EventHatcheryDeleted    EventType = "HatcheryDeleted"

	EventRepositoryCreated EventType = "RepositoryCreated"
	EventRepositoryDeleted EventType = "RepositoryDeleted"

	EventOrganizationCreated EventType = "OrganizationCreated"
	EventOrganizationDeleted EventType = "OrganizationDeleted"

	EventRegionCreated EventType = "RegionCreated"
	EventRegionDeleted EventType = "RegionDeleted"

	EventPermissionCreated EventType = "PermissionCreated"
	EventPermissionUpdated EventType = "PermissionUpdated"
	EventPermissionDeleted EventType = "PermissionDeleted"

	EventUserCreated       EventType = "UserCreated"
	EventUserUpdated       EventType = "UserUpdated"
	EventUserDeleted       EventType = "UserDeleted"
	EventUserGPGKeyCreated EventType = "UserGPGKeyCreated"
	EventUserGPGKeyDeleted EventType = "UserGPGKeyDeleted"

	EventPluginCreated EventType = "PluginCreated"
	EventPluginUpdated EventType = "PluginUpdated"
	EventPluginDeleted EventType = "PluginDeleted"

	EventIntegrationModelCreated EventType = "IntegrationModelCreated"
	EventIntegrationModelUpdated EventType = "IntegrationModelUpdated"
	EventIntegrationModelDeleted EventType = "IntegrationModelDeleted"

	EventIntegrationCreated EventType = "IntegrationCreated"
	EventIntegrationUpdated EventType = "IntegrationUpdated"
	EventIntegrationDeleted EventType = "IntegrationDeleted"

	EventProjectCreated EventType = "ProjectCreated"
	EventProjectUpdated EventType = "ProjectUpdated"
	EventProjectDeleted EventType = "ProjectDeleted"

	EventNotificationCreated EventType = "NotificationCreated"
	EventNotificationUpdated EventType = "NotificationUpdated"
	EventNotificationDeleted EventType = "NotificationDeleted"

	EventVariableSetCreated     EventType = "VariableSetCreated"
	EventVariableSetDeleted     EventType = "VariableSetDeleted"
	EventVariableSetItemCreated EventType = "VariableSetItemCreated"
	EventVariableSetItemUpdated EventType = "VariableSetItemUpdated"
	EventVariableSetItemDeleted EventType = "VariableSetItemDeleted"
)

// FullEventV2 uses to process event
type FullEventV2 struct {
	ID               string          `json:"id"`
	Type             EventType       `json:"type"`
	Payload          json.RawMessage `json:"payload"`
	ProjectKey       string          `json:"project_key,omitempty"`
	VCSName          string          `json:"vcs_name,omitempty"`
	Repository       string          `json:"repository,omitempty"`
	Workflow         string          `json:"workflow,omitempty"`
	WorkflowRunID    string          `json:"workflow_run_id,omitempty"`
	RunJobID         string          `json:"run_job_id,omitempty"`
	RunNumber        int64           `json:"run_number,omitempty"`
	RunAttempt       int64           `json:"run_attempt,omitempty"`
	Region           string          `json:"region,omitempty"`
	Hatchery         string          `json:"hatchery,omitempty"`
	ModelType        string          `json:"model_type,omitempty"`
	JobID            string          `json:"job_id,omitempty"`
	Status           string          `json:"status,omitempty"`
	UserID           string          `json:"user_id,omitempty"`
	Username         string          `json:"username,omitempty"`
	RunResult        string          `json:"run_result,omitempty"`
	Entity           string          `json:"entity,omitempty"`
	Organization     string          `json:"organization,omitempty"`
	Permission       string          `json:"permission,omitempty"`
	Plugin           string          `json:"plugin,omitempty"`
	GPGKey           string          `json:"gpg_key,omitempty"`
	IntegrationModel string          `json:"integration_model,omitempty"`
	Integration      string          `json:"integration,omitempty"`
	KeyName          string          `json:"key_name,omitempty"`
	KeyType          string          `json:"key_type,omitempty"`
	Variable         string          `json:"variable,omitempty"`
	Notification     string          `json:"notification,omitempty"`
	VariableSet      string          `json:"variable_set,omitempty"`
	Item             string          `json:"item,omitempty"`
	Timestamp        time.Time       `json:"timestamp"`
}

type GlobalEventV2 struct {
	ID        string          `json:"id"`
	Type      EventType       `json:"type"`
	Payload   json.RawMessage `json:"payload"`
	Timestamp time.Time       `json:"timestamp"`
}

type ProjectEventV2 struct {
	ProjectKey string `json:"project_key"`
}

type AnalysisEvent struct {
	GlobalEventV2
	ProjectEventV2
	VCSName    string `json:"vcs_name"`
	Repository string `json:"repository"`
	Status     string `json:"status"`
	UserID     string `json:"user_id,omitempty"`
	Username   string `json:"username,omitempty"`
}

type EntityEvent struct {
	GlobalEventV2
	ProjectEventV2
	VCSName    string `json:"vcs_name"`
	Repository string `json:"repository"`
	Entity     string `json:"entity"`
	UserID     string `json:"user_id,omitempty"`
	Username   string `json:"username,omitempty"`
}

type HatcheryEvent struct {
	GlobalEventV2
	Hatchery string `json:"hatchery"`
	UserID   string `json:"user_id,omitempty"`
	Username string `json:"username,omitempty"`
}

type OrganizationEvent struct {
	GlobalEventV2
	Organization string `json:"organization"`
	UserID       string `json:"user_id"`
	Username     string `json:"username"`
}

type PermissionEvent struct {
	GlobalEventV2
	Permission string `json:"permission"`
	UserID     string `json:"user_id"`
	Username   string `json:"username"`
}

type PluginEvent struct {
	GlobalEventV2
	Plugin   string `json:"plugin"`
	UserID   string `json:"user_id"`
	Username string `json:"username"`
}

type RegionEvent struct {
	GlobalEventV2
	Region   string `json:"region"`
	UserID   string `json:"user_id"`
	Username string `json:"username"`
}

type RepositoryEvent struct {
	GlobalEventV2
	ProjectEventV2
	VCSName    string `json:"vcs_name"`
	Repository string `json:"repository"`
	UserID     string `json:"user_id"`
	Username   string `json:"username"`
}

type VCSEvent struct {
	GlobalEventV2
	ProjectEventV2
	VCSName  string `json:"vcs_name"`
	UserID   string `json:"user_id"`
	Username string `json:"username"`
}

type KeyEvent struct {
	GlobalEventV2
	ProjectEventV2
	KeyName  string `json:"key_name"`
	KeyType  string `json:"key_type"`
	UserID   string `json:"user_id"`
	Username string `json:"username"`
}

type VariableEvent struct {
	GlobalEventV2
	ProjectEventV2
	Variable string `json:"variable"`
	UserID   string `json:"user_id"`
	Username string `json:"username"`
}

type IntegrationModelEvent struct {
	GlobalEventV2
	IntegrationModel string `json:"integration_model"`
	UserID           string `json:"user_id"`
	Username         string `json:"username"`
}

type ProjectIntegrationEvent struct {
	GlobalEventV2
	ProjectEventV2
	Integration      string `json:"integration"`
	IntegrationModel string `json:"integration_model"`
	UserID           string `json:"user_id"`
	Username         string `json:"username"`
}

type NotificationEvent struct {
	GlobalEventV2
	ProjectEventV2
	Notification string `json:"notification"`
	UserID       string `json:"user_id"`
	Username     string `json:"username"`
}

type ProjectVariableSetEvent struct {
	GlobalEventV2
	ProjectEventV2
	VariableSet string `json:"variable_set"`
	UserID      string `json:"user_id"`
	Username    string `json:"username"`
}

type ProjectVariableSetItemEvent struct {
	GlobalEventV2
	ProjectEventV2
	VariableSet string `json:"variable_set"`
	Item        string `json:"item"`
	UserID      string `json:"user_id"`
	Username    string `json:"username"`
}

type UserEvent struct {
	GlobalEventV2
	UserID   string `json:"user_id"`
	Username string `json:"username"`
}

type UserGPGEvent struct {
	GlobalEventV2
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	GPGKey   string `json:"gpg_key"`
}

type WorkflowRunEvent struct {
	GlobalEventV2
	ProjectEventV2
	VCSName       string              `json:"vcs_name"`
	Repository    string              `json:"repository"`
	Workflow      string              `json:"workflow"`
	RunNumber     int64               `json:"run_number"`
	RunAttempt    int64               `json:"run_attempt"`
	Status        V2WorkflowRunStatus `json:"status"`
	WorkflowRunID string              `json:"workflow_run_id"`
	UserID        string              `json:"user_id"`
	Username      string              `json:"username"`
}

type WorkflowRunJobEvent struct {
	GlobalEventV2
	ProjectEventV2
	VCSName       string                 `json:"vcs_name"`
	Repository    string                 `json:"repository"`
	Workflow      string                 `json:"workflow"`
	WorkflowRunID string                 `json:"workflow_run_id"`
	RunJobID      string                 `json:"run_job_id"`
	RunNumber     int64                  `json:"run_number"`
	RunAttempt    int64                  `json:"run_attempt"`
	Region        string                 `json:"region"`
	Hatchery      string                 `json:"hatchery"`
	ModelType     string                 `json:"model_type"`
	JobID         string                 `json:"job_id"`
	Status        V2WorkflowRunJobStatus `json:"status"`
	UserID        string                 `json:"user_id"`
	Username      string                 `json:"username"`
}

type WorkflowRunJobManualEvent struct {
	GlobalEventV2
	ProjectEventV2
	VCSName       string              `json:"vcs_name"`
	Repository    string              `json:"repository"`
	Workflow      string              `json:"workflow"`
	JobID         string              `json:"job_id"`
	RunNumber     int64               `json:"run_number"`
	RunAttempt    int64               `json:"run_attempt"`
	Status        V2WorkflowRunStatus `json:"status"`
	WorkflowRunID string              `json:"workflow_run_id"`
	UserID        string              `json:"user_id"`
	Username      string              `json:"username"`
}

type WorkflowRunJobRunResultEvent struct {
	GlobalEventV2
	ProjectEventV2
	VCSName       string `json:"vcs_name"`
	Repository    string `json:"repository"`
	Workflow      string `json:"workflow"`
	WorkflowRunID string `json:"workflow_run_id"`
	RunJobID      string `json:"run_job_id"`
	RunNumber     int64  `json:"run_number"`
	RunAttempt    int64  `json:"run_attempt"`
	Region        string `json:"region"`
	Hatchery      string `json:"hatchery"`
	ModelType     string `json:"model_type"`
	JobID         string `json:"job_id"`
	Status        string `json:"status"`
	UserID        string `json:"user_id"`
	Username      string `json:"username"`
	RunResult     string `json:"run_result"`
}

type ProjectEvent struct {
	GlobalEventV2
	ProjectEventV2
	UserID   string `json:"user_id"`
	Username string `json:"username"`
}

func NewEventWorkflowRunPayload(wr V2WorkflowRun, rjs map[string]V2WorkflowRunJob, runResults []V2WorkflowRunResult) (*EventWorkflowRunPayload, error) {
	p := EventWorkflowRunPayload{
		ID:           wr.ID,
		ProjectKey:   wr.ProjectKey,
		VCSServer:    wr.VCSServer,
		Repository:   wr.Repository,
		WorkflowName: wr.WorkflowName,
		WorkflowSha:  wr.WorkflowSha,
		WorkflowRef:  wr.WorkflowRef,
		Status:       wr.Status,
		RunNumber:    wr.RunNumber,
		RunAttempt:   wr.RunAttempt,
		Started:      wr.Started,
		LastModified: wr.LastModified,

		WorkflowData: wr.WorkflowData,
		UserID:       wr.UserID,
		Username:     wr.Username,
		AdminMFA:     wr.AdminMFA,
		RunEvent:     wr.RunEvent,
		RunJobEvent:  wr.RunJobEvent,
		Annotations:  wr.Annotations,
		Contexts: EventWorkflowRunPayloadContexts{
			CDS: wr.Contexts.CDS,
			Git: wr.Contexts.Git,
			Env: wr.Contexts.Env,
		},
	}
	p.Contexts.Jobs = JobsResultContext{}
	for _, rj := range rjs {
		jobResult := JobResultContext{
			Result:  rj.Status,
			Outputs: JobResultOutput{},
		}
		for _, r := range runResults {
			if rj.ID != r.WorkflowRunJobID {
				continue
			}
			switch r.Type {
			case V2WorkflowRunResultTypeVariable, V2WorkflowRunResultVariableDetailType:
				x, err := GetConcreteDetail[*V2WorkflowRunResultVariableDetail](&r)
				if err != nil {
					return nil, err
				}
				jobResult.Outputs[x.Name] = x.Value
			default:
				if jobResult.JobRunResults == nil {
					jobResult.JobRunResults = JobRunResults{}
				}
				jobResult.JobRunResults[r.Name()], _ = r.GetDetail()
			}
		}
		p.Contexts.Jobs[rj.JobID] = jobResult
	}
	return &p, nil
}

type EventWorkflowRunPayload struct {
	ID           string                 `json:"id"`
	ProjectKey   string                 `json:"project_key"`
	VCSServer    string                 `json:"vcs_server"`
	Repository   string                 `json:"repository"`
	WorkflowName string                 `json:"workflow_name"`
	WorkflowSha  string                 `json:"workflow_sha"`
	WorkflowRef  string                 `json:"workflow_ref"`
	Status       V2WorkflowRunStatus    `json:"status"`
	RunNumber    int64                  `json:"run_number"`
	RunAttempt   int64                  `json:"run_attempt"`
	Started      time.Time              `json:"started"`
	LastModified time.Time              `json:"last_modified"`
	ToDelete     bool                   `json:"to_delete"`
	WorkflowData V2WorkflowRunData      `json:"workflow_data"`
	UserID       string                 `json:"user_id"`
	Username     string                 `json:"username"`
	AdminMFA     bool                   `json:"admin_mfa"`
	RunEvent     V2WorkflowRunEvent     `json:"event,omitempty"`
	RunJobEvent  V2WorkflowRunJobEvents `json:"job_events,omitempty"`
	Annotations  WorkflowRunAnnotations `json:"annotations,omitempty"`

	Contexts EventWorkflowRunPayloadContexts `json:"contexts" db:"contexts"`
}

type EventWorkflowRunPayloadContexts struct {
	CDS  CDSContext        `json:"cds,omitempty"`
	Git  GitContext        `json:"git,omitempty"`
	Env  map[string]string `json:"env,omitempty"`
	Jobs JobsResultContext `json:"jobs,omitempty"`
}

type EventWorkflowRunPayloadContextEvent struct {
	Status   string `json:"status,omitempty"`
	AdminMFA bool   `json:"admin_mfa"`
}
