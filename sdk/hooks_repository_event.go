package sdk

import (
	"fmt"

	"crypto/sha512"
	"encoding/base64"

	"golang.org/x/crypto/pbkdf2"
)

const (
	SignHeaderVCSName   = "X-Cds-Hooks-Vcs-Name"
	SignHeaderRepoName  = "X-Cds-Hooks-Repo-Name"
	SignHeaderVCSType   = "X-Cds-Hooks-Vcs-Type"
	SignHeaderEventName = "X-Cds-Hooks-Event-Name"

	WorkflowHookEventWorkflowUpdate = "workflow-update"
	WorkflowHookEventModelUpdate    = "model-update"
	WorkflowHookEventPush           = "push"
	WorkflowHookManual              = "manual"

	WorkflowHookEventPullRequest             = "pull-request"
	WorkflowHookEventPullRequestTypeOpened   = "opened"
	WorkflowHookEventPullRequestTypeReopened = "reopened"
	WorkflowHookEventPullRequestTypeClosed   = "closed"
	WorkflowHookEventPullRequestTypeEdited   = "edited"

	WorkflowHookEventPullRequestComment            = "pull-request-comment"
	WorkflowHookEventPullRequestCommentTypeCreated = "created"
	WorkflowHookEventPullRequestCommentTypeDeleted = "deleted"
	WorkflowHookEventPullRequestCommentTypeEdited  = "edited"

	RepoEventPush = "push"

	HookEventStatusScheduled     = "Scheduled"
	HookEventStatusAnalysis      = "Analyzing"
	HookEventStatusWorkflowHooks = "WorkflowHooks"
	HookEventStatusSignKey       = "SignKey"
	HookEventStatusWorkflow      = "Workflow"
	HookEventStatusDone          = "Done"
	HookEventStatusError         = "Error"
	HookEventStatusSkipped       = "Skipped"

	HookEventWorkflowStatusScheduler = "Scheduled"
	HookEventWorkflowStatusDone      = "Done"
)

type HookEventCallback struct {
	AnalysisCallback   *HookAnalysisCallback  `json:"analysis_callback"`
	SigningKeyCallback *HookSigninKeyCallback `json:"signing_key_callback"`
	HookEventUUID      string                 `json:"hook_event_uuid"`
	VCSServerType      string                 `json:"vcs_server_type"`
	VCSServerName      string                 `json:"vcs_server_name"`
	RepositoryName     string                 `json:"repository_name"`
}

type HookSigninKeyCallback struct {
	Status        OperationStatus `json:"status"`
	SignKey       string          `json:"sign_key"`
	SemverCurrent string          `json:"semver_current"`
	SemverNext    string          `json:"semver_next"`
	Error         string          `json:"error"`
}

type HookAnalysisCallback struct {
	AnalysisID     string           `json:"analysis_id"`
	AnalysisStatus string           `json:"analysis_status"`
	Models         []EntityFullName `json:"models"`
	Workflows      []EntityFullName `json:"workflows"`
}

type HookRepository struct {
	VCSServerType  string `json:"vcs_server_type"`
	VCSServerName  string `json:"vcs_server_name" cli:"vcs_server_name"`
	RepositoryName string `json:"repository_name" cli:"repository_name"`
	Stopped        bool   `json:"stopped" cli:"stopped"`
}

type HookRepositoryEvent struct {
	UUID                      string                         `json:"uuid"`
	Created                   int64                          `json:"created"`
	EventName                 string                         `json:"event_name"` // WorkflowHookEventPush, sdk.WorkflowHookEventPullRequest
	EventType                 string                         `json:"event_type"` // created, deleted, edited, opened
	VCSServerType             string                         `json:"vcs_server_type"`
	VCSServerName             string                         `json:"vcs_server_name"`
	RepositoryName            string                         `json:"repository_name"`
	Body                      []byte                         `json:"body"`
	ExtractData               HookRepositoryEventExtractData `json:"extracted_data"`
	Status                    string                         `json:"status"`
	ProcessingTimestamp       int64                          `json:"processing_timestamp"`
	LastUpdate                int64                          `json:"last_update"`
	LastError                 string                         `json:"last_error"`
	NbErrors                  int64                          `json:"nb_errors"`
	Analyses                  []HookRepositoryEventAnalysis  `json:"analyses"`
	ModelUpdated              []EntityFullName               `json:"model_updated"`
	WorkflowUpdated           []EntityFullName               `json:"workflow_updated"`
	WorkflowHooks             []HookRepositoryEventWorkflow  `json:"workflows"`
	UserID                    string                         `json:"user_id"`
	Username                  string                         `json:"username"`
	SignKey                   string                         `json:"sign_key"`
	SigningKeyOperation       string                         `json:"signing_key_operation"`
	SigningKeyOperationStatus OperationStatus                `json:"signing_key_operation_status"`
	SemverCurrent             string                         `json:"semver_current"`
	SemverNext                string                         `json:"semver_next"`
}

type HookRepositoryEventWorkflow struct {
	ProjectKey           string `json:"project_key"`
	VCSIdentifier        string `json:"vcs_identifier"`
	RepositoryIdentifier string `json:"repository_identifier"`
	WorkflowName         string `json:"workflow_name"`
	EntityID             string `json:"entity_id"`
	Ref                  string `json:"ref"`
	Type                 string `json:"type"`
	Status               string `json:"status"`
	TargetBranch         string `json:"target_branch,omitempty"`
	TargetCommit         string `json:"target_commit,omitempty"`
	ModelFullName        string `json:"model,omitempty"`
	RunID                string `json:"run_id,omitempty"`
}

type HookRepositoryEventExtractData struct {
	CDSEventName   string   `json:"cds_event_name"`
	CDSEventType   string   `json:"cds_event_type"`
	Commit         string   `json:"commit"`
	Paths          []string `json:"paths"`
	Ref            string   `json:"ref"`
	ProjectManual  string   `json:"manual_project"`
	WorkflowManual string   `json:"manual_workflow"`
}

type GenerateRepositoryWebhook struct {
	Key string `json:"key"`
}

func (h *HookRepositoryEvent) GetFullName() string {
	return fmt.Sprintf("%s/%s/%s/%s", h.VCSServerType, h.VCSServerName, h.RepositoryName, h.UUID)
}

type HookRepositoryEventAnalysis struct {
	AnalyzeID  string `json:"analyze_id"`
	Status     string `json:"status"`
	ProjectKey string `json:"project_key"`
}

type HookRetrieveSignKeyRequest struct {
	ProjectKey     string `json:"projectKey"`
	VCSServerType  string `json:"vcs_server_type"`
	VCSServerName  string `json:"vcs_server_name"`
	RepositoryName string `json:"repository_name"`
	Commit         string `json:"commit"`
	Ref            string `json:"ref"`
	HookEventUUID  string `json:"hook_event_uuid"`
}

type HookRetrieveUserRequest struct {
	ProjectKey     string `json:"projectKey"`
	VCSServerType  string `json:"vcs_server_type"`
	VCSServerName  string `json:"vcs_server_name"`
	RepositoryName string `json:"repository_name"`
	Commit         string `json:"commit"`
	SignKey        string `json:"sign_key"`
	HookEventUUID  string `json:"hook_event_uuid"`
}

type HookRetrieveUserResponse struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
}

type AnalysisRequest struct {
	ProjectKey    string `json:"projectKey"`
	VcsName       string `json:"vcsName"`
	RepoName      string `json:"repoName"`
	Ref           string `json:"ref"`
	Commit        string `json:"commit"`
	HookEventUUID string `json:"hook_event_uuid"`
}

type AnalysisResponse struct {
	AnalysisID string `json:"analysis_id" cli:"analysis_id"`
	Status     string `json:"status" cli:"status"`
}

func GenerateRepositoryWebHookSecret(hookSecretKey, vcsName, repoName string) string {
	pass := fmt.Sprintf("%s-%s", vcsName, repoName)
	keyBytes := pbkdf2.Key([]byte(pass), []byte(hookSecretKey), 4096, 128, sha512.New)
	key64 := base64.StdEncoding.EncodeToString(keyBytes)
	return key64
}
