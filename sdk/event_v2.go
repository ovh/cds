package sdk

import (
	"encoding/json"
)

const (
	EventAnalysisStart = "AnalysisStart"
	EventAnalysisDone  = "AnalysisDone"

	EventRunJobEnqueued         = "RunJobEnqueued"
	EventRunJobScheduled        = "RunJobScheduled"
	EventRunJobBuilding         = "RunJobBuilding"
	EventRunJobManualTriggered  = "RunJobManualTriggered"
	EventRunJobRunResultAdded   = "RunJobRunResultAdded"
	EventRunJobRunResultUpdated = "RunJobRunResultUpdated"
	EventRunJobEnded            = "RunJobEnded"

	EventRunCrafted          = "RunCrafted"
	EventRunBuilding         = "RunBuilding"
	EventRunEnded            = "RunEnded"
	EventRunRestartFailedJob = "RunRestartFailedJob"

	EventEntityCreated = "EntityCreated"
	EventEntityUpdated = "EntityUpdated"
	EventEntityDeleted = "EntityDeleted"

	EventVCSCreated = "VCSCreated"
	EventVCSUpdated = "VCSUpdated"
	EventVCSDeleted = "VCSDeleted"

	EventHatcheryCreated = "HatcheryCreated"
	EventHatcheryUpdated = "HatcheryUpdated"
	EventHatcheryDeleted = "HatcheryDeleted"

	EventRepositoryCreated = "RepositoryCreated"
	EventRepositoryDeleted = "RepositoryDeleted"

	EventOrganizationCreated = "OrganizationCreated"
	EventOrganizationDeleted = "OrganizationDeleted"

	EventRegionCreated = "RegionCreated"
	EventRegionDeleted = "RegionDeleted"

	EventPermissionCreated = "PermissionCreated"
	EventPermissionUpdated = "PermissionUpdated"
	EventPermissionDeleted = "PermissionDeleted"

	EventUserCreated       = "UserCreated"
	EventUserUpdated       = "UserUpdated"
	EventUserDeleted       = "UserDeleted"
	EventUserGPGKeyCreated = "UserGPGKeyCreated"
	EventUserGPGKeyDeleted = "UserGPGKeyDeleted"

	EventPluginCreated = "PluginCreated"
	EventPluginUpdated = "PluginUpdated"
	EventPluginDeleted = "PluginDeleted"

	EventIntegrationModelCreated = "IntegrationModelCreated"
	EventIntegrationModelUpdated = "IntegrationModelUpdated"
	EventIntegrationModelDeleted = "IntegrationModelDeleted"

	EventIntegrationCreated = "IntegrationCreated"
	EventIntegrationUpdated = "IntegrationUpdated"
	EventIntegrationDeleted = "IntegrationDeleted"

	EventKeyCreated = "KeyCreated"
	EventKeyDeleted = "KeyDeleted"

	EventVariableCreated = "VariableCreated"
	EventVariableUpdated = "VariableUpdated"
	EventVariableDeleted = "VariableDeleted"

	EventProjectCreated = "ProjectCreated"
	EventProjectUpdated = "ProjectUpdated"
	EventProjectDeleted = "ProjectDeleted"
)

// FullEventV2 uses to process event
type FullEventV2 struct {
	ID               string          `json:"id"`
	Type             string          `json:"type"`
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
}

type GlobalEventV2 struct {
	ID      string          `json:"id"`
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

type ProjectEventV2 struct {
	ID         string          `json:"id"`
	Type       string          `json:"type"`
	ProjectKey string          `json:"project_key"`
	Payload    json.RawMessage `json:"payload"`
}

type AnalysisEvent struct {
	ProjectEventV2
	ProjectKey string `json:"project_key"`
	VCSName    string `json:"vcs_name"`
	Repository string `json:"repository"`
	Status     string `json:"status"`
	UserID     string `json:"user_id,omitempty"`
	Username   string `json:"username,omitempty"`
}

type EntityEvent struct {
	ProjectEventV2
	ProjectKey string `json:"project_key"`
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
	ProjectEventV2
	VCSName    string `json:"vcs_name"`
	Repository string `json:"repository"`
	UserID     string `json:"user_id"`
	Username   string `json:"username"`
}

type VCSEvent struct {
	ProjectEventV2
	VCSName  string `json:"vcs_name"`
	UserID   string `json:"user_id"`
	Username string `json:"username"`
}

type KeyEvent struct {
	ProjectEventV2
	KeyName  string `json:"key_name"`
	KeyType  string `json:"key_type"`
	UserID   string `json:"user_id"`
	Username string `json:"username"`
}

type VariableEvent struct {
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
	ProjectEventV2
	Integration      string `json:"integration"`
	IntegrationModel string `json:"integration_model"`
	UserID           string `json:"user_id"`
	Username         string `json:"username"`
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
	ProjectEventV2
	VCSName       string `json:"vcs_name"`
	Repository    string `json:"repository"`
	Workflow      string `json:"workflow"`
	RunNumber     int64  `json:"run_number"`
	RunAttempt    int64  `json:"run_attempt"`
	Status        string `json:"status"`
	WorkflowRunID string `json:"workflow_run_id"`
	UserID        string `json:"user_id"`
	Username      string `json:"username"`
}

type WorkflowRunJobEvent struct {
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
}

type WorkflowRunJobManualEvent struct {
	ProjectEventV2
	VCSName       string `json:"vcs_name"`
	Repository    string `json:"repository"`
	Workflow      string `json:"workflow"`
	JobID         string `json:"job_id"`
	RunNumber     int64  `json:"run_number"`
	RunAttempt    int64  `json:"run_attempt"`
	Status        string `json:"status"`
	WorkflowRunID string `json:"workflow_run_id"`
	UserID        string `json:"user_id"`
	Username      string `json:"username"`
}

type WorkflowRunJobRunResultEvent struct {
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
	ProjectEventV2
	UserID   string `json:"user_id"`
	Username string `json:"username"`
}
