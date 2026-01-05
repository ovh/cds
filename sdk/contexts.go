package sdk

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

// DEPRECATED - Only use on old workflow
type NodeRunContext struct {
	Git  GitContext        `json:"git,omitempty"`
	Vars map[string]string `json:"vars,omitempty"`
	Jobs JobsResultContext `json:"jobs,omitempty"`
}

func (m NodeRunContext) Value() (driver.Value, error) {
	j, err := json.Marshal(m)
	return j, WrapError(err, "cannot marshal RunContext")
}

func (m *NodeRunContext) Scan(src interface{}) error {
	if src == nil {
		return nil
	}
	source, ok := src.([]byte)
	if !ok {
		return WithStack(fmt.Errorf("type assertion .([]byte]) failed (%T)", src))
	}
	return WrapError(JSONUnmarshal(source, m), "cannot unmarshal RunContext")
}

type JobRunContext struct {
	NodeRunContext
	Job     JobContext        `json:"job,omitempty"`
	Secrets map[string]string `json:"secrets,omitempty"`
}

func (m JobRunContext) Value() (driver.Value, error) {
	j, err := json.Marshal(m)
	return j, WrapError(err, "cannot marshal JobRunContext")
}

func (m *JobRunContext) Scan(src interface{}) error {
	if src == nil {
		return nil
	}
	source, ok := src.([]byte)
	if !ok {
		return WithStack(fmt.Errorf("type assertion .([]byte) failed (%T)", src))
	}
	return WrapError(JSONUnmarshal(source, m), "cannot unmarshal JobRunContext")
}

type ActionContext struct {
	Inputs map[string]interface{} `json:"inputs,omitempty"`
}

type CDSContext struct {
	// Workflow
	EventName          WorkflowHookEventName  `json:"event_name,omitempty" jsonschema:"example=push" jsonschema_description:"Name of the event that triggered the workflow"`
	Event              map[string]interface{} `json:"event,omitempty" jsonschema_description:"Full event payload that triggered the workflow"`
	ProjectKey         string                 `json:"project_key,omitempty" jsonschema:"example=MYPROJECT" jsonschema_description:"Project key"`
	RunID              string                 `json:"run_id,omitempty" jsonschema:"example=550e8400-e29b-41d4-a716-446655440000" jsonschema_description:"Unique identifier of the workflow run"`
	RunNumber          int64                  `json:"run_number,omitempty" jsonschema:"example=42" jsonschema_description:"Sequential number of the workflow run"`
	RunAttempt         int64                  `json:"run_attempt,omitempty" jsonschema:"example=1" jsonschema_description:"Attempt number if the run was restarted"`
	RunURL             string                 `json:"run_url,omitempty" jsonschema:"example=https://cds.example.com/project/MYPROJECT/run/42" jsonschema_description:"URL to view the workflow run in CDS UI"`
	Workflow           string                 `json:"workflow,omitempty" jsonschema:"example=my-workflow" jsonschema_description:"Name of the workflow"`
	WorkflowRef        string                 `json:"workflow_ref,omitempty" jsonschema:"example=refs/heads/main" jsonschema_description:"Git reference of the workflow file"`
	WorkflowSha        string                 `json:"workflow_sha,omitempty" jsonschema:"example=a1b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6q7r8s9t0" jsonschema_description:"Git commit SHA of the workflow file"`
	WorkflowVCSServer  string                 `json:"workflow_vcs_server,omitempty" jsonschema:"example=github" jsonschema_description:"VCS server name for the workflow repository"`
	WorkflowRepository string                 `json:"workflow_repository,omitempty" jsonschema:"example=ovh/cds" jsonschema_description:"Repository name for the workflow"`
	TriggeringActor    string                 `json:"triggering_actor,omitempty" jsonschema:"example=john.doe" jsonschema_description:"User or system that triggered the workflow"`
	Version            string                 `json:"version,omitempty" jsonschema:"example=1.2.3" jsonschema_description:"Current semantic version"`
	VersionNext        string                 `json:"version_next,omitempty" jsonschema:"example=1.2.4" jsonschema_description:"Next semantic version"`

	// Workflow Template
	WorkflowTemplate                 string            `json:"workflow_template,omitempty" jsonschema:"example=my-template" jsonschema_description:"Name of the workflow template"`
	WorkflowTemplateRef              string            `json:"workflow_template_ref,omitempty" jsonschema:"example=refs/heads/main" jsonschema_description:"Git reference of the workflow template"`
	WorkflowTemplateSha              string            `json:"workflow_template_sha,omitempty" jsonschema:"example=a1b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6q7r8s9t0" jsonschema_description:"Git commit SHA of the workflow template"`
	WorkflowTemplateVCSServer        string            `json:"workflow_template_vcs_server,omitempty" jsonschema:"example=github" jsonschema_description:"VCS server name for the template repository"`
	WorkflowTemplateRepository       string            `json:"workflow_template_repository,omitempty" jsonschema:"example=ovh/cds-templates" jsonschema_description:"Repository name for the template"`
	WorkflowTemplateProjectKey       string            `json:"workflow_template_project_key,omitempty" jsonschema:"example=TEMPLATES" jsonschema_description:"Project key of the template"`
	WorkflowTemplateParams           map[string]string `json:"workflow_template_params,omitempty" jsonschema_description:"Parameters passed to the workflow template"`
	WorkflowTemplateCommitWebURL     string            `json:"workflow_template_commit_web_url,omitempty" jsonschema:"example=https://github.com/ovh/cds-templates/commit/a1b2c3d4" jsonschema_description:"Web URL to view the template commit"`
	WorkflowTemplateRefWebURL        string            `json:"workflow_template_ref_web_url,omitempty" jsonschema:"example=https://github.com/ovh/cds-templates/tree/main" jsonschema_description:"Web URL to view the template reference"`
	WorkflowTemplateRepositoryWebURL string            `json:"workflow_template_repository_web_url,omitempty" jsonschema:"example=https://github.com/ovh/cds-templates" jsonschema_description:"Web URL to view the template repository"`

	// Job
	Job   string `json:"job,omitempty" jsonschema:"example=build" jsonschema_description:"Name of the current job"`
	Stage string `json:"stage,omitempty" jsonschema:"example=build-stage" jsonschema_description:"Name of the current stage"`

	// Worker
	Workspace string `json:"workspace,omitempty" jsonschema:"example=/workspace" jsonschema_description:"Absolute path to the workspace directory"`
}

type GitContext struct {
	Server               string   `json:"server,omitempty" jsonschema:"example=github" jsonschema_description:"VCS server name"`
	Repository           string   `json:"repository,omitempty" jsonschema:"example=ovh/cds" jsonschema_description:"Repository identifier"`
	RepositoryOrigin     string   `json:"repository_origin",omitempty jsonschema:"example=fork-user/cds" jsonschema_description:"Origin repository for pull requests"`
	RepositoryURL        string   `json:"repositoryUrl,omitempty" jsonschema:"example=https://github.com/ovh/cds.git" jsonschema_description:"Git clone URL"`
	RepositoryWebURL     string   `json:"repository_web_url,omitempty" jsonschema:"example=https://github.com/ovh/cds" jsonschema_description:"Web URL of the repository"`
	RefWebURL            string   `json:"ref_web_url,omitempty" jsonschema:"example=https://github.com/ovh/cds/tree/main" jsonschema_description:"Web URL of the reference"`
	CommitWebURL         string   `json:"commit_web_url,omitempty" jsonschema:"example=https://github.com/ovh/cds/commit/a1b2c3d4e5f6" jsonschema_description:"Web URL of the commit"`
	CommitMessage        string   `json:"commit_message,omitempty" jsonschema:"example=feat: add new feature" jsonschema_description:"Commit message"`
	Author               string   `json:"author,omitempty" jsonschema:"example=John Doe" jsonschema_description:"Commit author name"`
	AuthorEmail          string   `json:"author_email,omitempty" jsonschema:"example=john.doe@example.com" jsonschema_description:"Commit author email"`
	Ref                  string   `json:"ref,omitempty" jsonschema:"example=refs/heads/main" jsonschema_description:"Full git reference"`
	RefName              string   `json:"ref_name,omitempty" jsonschema:"example=main" jsonschema_description:"Short reference name"`
	Sha                  string   `json:"sha,omitempty" jsonschema:"example=a1b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6q7r8s9t0" jsonschema_description:"Full commit SHA"`
	ShaShort             string   `json:"sha_short,omitempty" jsonschema:"example=a1b2c3d" jsonschema_description:"Short commit SHA (7 characters)"`
	RefType              string   `json:"ref_type,omitempty" jsonschema:"example=branch" jsonschema_description:"Type of reference (branch, tag)"`
	Connection           string   `json:"connection,omitempty" jsonschema:"example=my-github-connection" jsonschema_description:"VCS connection name"`
	SSHKey               string   `json:"ssh_key,omitempty" jsonschema_description:"SSH private key for git operations"`
	Username             string   `json:"username,omitempty" jsonschema:"example=git-user" jsonschema_description:"Git username for authentication"`
	Token                string   `json:"token,omitempty" jsonschema_description:"Git access token for authentication"`
	SemverCurrent        string   `json:"semver_current,omitempty" jsonschema:"example=1.2.3" jsonschema_description:"Current semantic version"`
	SemverNext           string   `json:"semver_next,omitempty" jsonschema:"example=1.2.4" jsonschema_description:"Next semantic version"`
	ChangeSets           []string `json:"changesets,omitempty" jsonschema_description:"List of changed file paths"`
	PullRequestID        int64    `json:"pullrequest_id,omitempty" jsonschema:"example=123" jsonschema_description:"Pull request number"`
	PullRequestToRef     string   `json:"pullrequest_to_ref,omitempty" jsonschema:"example=refs/heads/main" jsonschema_description:"Target reference of the pull request"`
	PullRequestToRefName string   `json:"pullrequest_to_ref_name,omitempty" jsonschema:"example=main" jsonschema_description:"Target branch name of the pull request"`
	PullRequestWebURL    string   `json:"pullrequest_web_url,omitempty" jsonschema:"example=https://github.com/ovh/cds/pull/123" jsonschema_description:"Web URL of the pull request"`
	GPGKey               string   `json:"gpg_key,omitempty" jsonschema_description:"GPG private key for signing commits"`
	Email                string   `json:"email,omitempty" jsonschema:"example=git-user@example.com" jsonschema_description:"Git email for commits"`
}

type JobContext struct {
	// Update by worker
	Status string `json:"status"`

	// Set by hatchery
	Services map[string]JobContextService `json:"services"`
}

type JobContextService struct {
	ID   string            `json:"id"`
	Port map[string]string `json:"ports"`
}

type JobsResultContext map[string]JobResultContext

type JobsGateContext map[string]GateInputs

type JobResultContext struct {
	JobRunResults
	Result  V2WorkflowRunJobStatus `json:"result" jsonschema:"example=Success" jsonschema_description:"Final status of the job"`
	Outputs JobResultOutput        `json:"outputs" jsonschema_description:"Key-value map of output values defined by the job"`
}

type JobResultOutput map[string]any

type JobRunResults map[string]any

func (jro JobResultOutput) Value() (driver.Value, error) {
	j, err := json.Marshal(jro)
	return j, WrapError(err, "cannot marshal JobResultOutput")
}

func (jro *JobResultOutput) Scan(src interface{}) error {
	if src == nil {
		return nil
	}
	source, ok := src.(string)
	if !ok {
		return WithStack(fmt.Errorf("type assertion .(string) failed (%T)", src))
	}
	return WrapError(JSONUnmarshal([]byte(source), jro), "cannot unmarshal JobResultOutput")
}

type (
	StepsContext map[string]StepContext
	StepContext  struct {
		Conclusion V2WorkflowRunJobStatus `json:"conclusion" jsonschema:"example=Success" jsonschema_description:"Final status of the step after applying continue-on-error"` // result of a step after 'continue-on-error'
		Outcome    V2WorkflowRunJobStatus `json:"outcome" jsonschema:"example=Fail" jsonschema_description:"Actual status of the step before continue-on-error"`              // result of a step before 'continue-on-error'
		Outputs    JobResultOutput        `json:"outputs" jsonschema_description:"Key-value map of output values defined by the step using outputs in the step definition"`
	}
)

type (
	NeedsContext map[string]NeedContext
	NeedContext  struct {
		Result  V2WorkflowRunJobStatus `json:"result" jsonschema:"example=Success" jsonschema_description:"Final status of the needed job"`
		Outputs JobResultOutput        `json:"outputs" jsonschema_description:"Key-value map of output values from the needed job"`
	}
)

func (sc StepsContext) Value() (driver.Value, error) {
	j, err := json.Marshal(sc)
	return j, WrapError(err, "cannot marshal StepsContext")
}

func (sc *StepsContext) Scan(src interface{}) error {
	if src == nil {
		return nil
	}
	source, ok := src.(string)
	if !ok {
		return WithStack(fmt.Errorf("type assertion .(string) failed (%T)", src))
	}
	return WrapError(JSONUnmarshal([]byte(source), sc), "cannot unmarshal StepsContext")
}
