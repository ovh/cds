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

	WorkflowHookEventPush = "push"

	HookEventStatusScheduled     = "Scheduled"
	HookEventStatusAnalysis      = "Analyzing"
	HookEventStatusWorkflowHooks = "WorkflowHooks"
	HookEventStatusDone          = "Done"
	HookEventStatusError         = "Error"
	HookEventStatusSkipped       = "Skipped"

	HookEventWorkflowStatusScheduler = "Scheduled"
	HookEventWorkflowStatusDone      = "Done"
)

type HookAnalysisCallback struct {
	AnalysisID     string           `json:"analysis_id"`
	AnalysisStatus string           `json:"analysis_status"`
	VCSServerType  string           `json:"vcs_server_type"`
	VCSServerName  string           `json:"vcs_server_name"`
	RepositoryName string           `json:"repository_name"`
	HookEventUUID  string           `json:"hook_event_uuid"`
	Models         []EntityFullName `json:"models"`
	Workflows      []EntityFullName `json:"workflows"`
}

type HookRepository struct {
	VCSServerType  string `json:"vcs_server_type"`
	VCSServerName  string `json:"vcs_server_name"`
	RepositoryName string `json:"repository_name"`
	Stopped        bool   `json:"stopped" cli:"Stopped"`
}

type HookRepositoryEvent struct {
	UUID                string                         `json:"uuid"`
	Created             int64                          `json:"created"`
	EventName           string                         `json:"event_name"`
	VCSServerType       string                         `json:"vcs_server_type"`
	VCSServerName       string                         `json:"vcs_server_name"`
	RepositoryName      string                         `json:"repository_name"`
	Body                []byte                         `json:"body"`
	ExtractData         HookRepositoryEventExtractData `json:"extracted_data"`
	Status              string                         `json:"status"`
	ProcessingTimestamp int64                          `json:"processing_timestamp"`
	LastUpdate          int64                          `json:"last_update"`
	LastError           string                         `json:"last_error"`
	NbErrors            int64                          `json:"nb_errors"`
	Analyses            []HookRepositoryEventAnalysis  `json:"analyses"`
	ModelUpdated        []EntityFullName               `json:"model_updated"`
	WorkflowUpdated     []EntityFullName               `json:"workflow_updated"`
	WorkflowHooks       []HookRepositoryEventWorkflow  `json:"workflows"`
}

type HookRepositoryEventWorkflow struct {
	ProjectKey           string `json:"project_key"`
	VCSIdentifier        string `json:"vcs_identifier"`
	RepositoryIdentifier string `json:"repository_identifier"`
	WorkflowName         string `json:"workflow_name"`
	EntityID             string `json:"entity_id"`
	Branch               string `json:"branch"`
	Type                 string `json:"type"`
	Status               string `json:"status"`
	TargetBranch         string `json:"target_branch"`
}

type HookRepositoryEventExtractData struct {
	Branch string   `json:"branch"`
	Commit string   `json:"commit"`
	Paths  []string `json:"paths"`
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

type AnalysisRequest struct {
	ProjectKey    string `json:"projectKey"`
	VcsName       string `json:"vcsName"`
	RepoName      string `json:"repoName"`
	Branch        string `json:"branch"`
	Commit        string `json:"commit"`
	HookEventUUID string `json:"hook_event_uuid"`
}

type AnalysisResponse struct {
	AnalysisID string `json:"analysis_id"`
	Status     string `json:"status"`
}

func GenerateRepositoryWebHookSecret(hookSecretKey, vcsName, repoName string) string {
	pass := fmt.Sprintf("%s-%s", vcsName, repoName)
	keyBytes := pbkdf2.Key([]byte(pass), []byte(hookSecretKey), 4096, 128, sha512.New)
	key64 := base64.StdEncoding.EncodeToString(keyBytes)
	return key64
}
