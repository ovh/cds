package sdk

const (
	EventAnalysisStart = "AnalysisStart"
	EventAnalysisDone  = "AnalysisDone"

	EventRunJobEnqueued         = "RunJobEnqueued"
	EventRunJobScheduled        = "RunJobScheduled"
	EventRunJobBuilding         = "RunJobBuilding"
	EventRunJobRunResultAdded   = "RunJobRunResultAdded"
	EventRunJobRunResultUpdated = "RunJobRunResultUpdated"
	EventRunJobEnded            = "RunJobEnded"

	EventRunCrafted          = "RunCrafted"
	EventRunBuilding         = "RunBuilding"
	EventRunEnded            = "RunEnded"
	EventRunRestartFailedJob = "RunRestartFailedJob"
)

type EventV2 struct {
	ID            string      `json:"id"`
	ProjectKey    string      `json:"project_key,omitempty"`
	VCSName       string      `json:"vcs_name,omitempty"`
	Repository    string      `json:"repository,omitempty"`
	Workflow      string      `json:"workflow,omitempty"`
	RunNumber     int64       `json:"run_number,omitempty"`
	RunAttempt    int64       `json:"run_attempt,omitempty"`
	Hatchery      string      `json:"hatchery,omitempty"`
	Region        string      `json:"region,omitempty"`
	ModelType     string      `json:"model_type,omitempty"`
	WorkflowRunID string      `json:"workflow_run_id,omitempty"`
	RunJobID      string      `json:"run_job_id,omitempty"`
	JobID         string      `json:"job_id,omitempty"`
	RunResultName string      `json:"run_result_name,omitempty"`
	Type          string      `json:"type,omitempty"`
	Status        string      `json:"status"`
	Payload       interface{} `json:"payload,omitempty"`
}
