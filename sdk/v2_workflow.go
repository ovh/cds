package sdk

import (
	"context"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/gorhill/cronexpr"
	"github.com/rockbears/yaml"
	"github.com/xeipuuv/gojsonschema"
)

const (
	WorkflowHookTypeRepository  = "RepositoryWebHook"
	WorkflowHookTypeWorkerModel = "WorkerModelUpdate"
	WorkflowHookTypeWorkflow    = "WorkflowUpdate"
	WorkflowHookTypeManual      = "Manual"
	WorkflowHookTypeWebhook     = "Webhook"
	WorkflowHookTypeScheduler   = "Scheduler"
	WorkflowHookTypeWorkflowRun = "WorkflowRun"
)

type WorkflowSemverType string

var AvailableSemverType = []WorkflowSemverType{SemverTypeGit, SemverTypeHelm, SemverTypeCargo, SemverTypeNpm, SemverTypeYarn, SemverTypeFile, SemverTypePoetry, SemverTypeDebian}

const (
	SemverTypeGit    WorkflowSemverType = "git"
	SemverTypeHelm   WorkflowSemverType = "helm"
	SemverTypeCargo  WorkflowSemverType = "cargo"
	SemverTypeNpm    WorkflowSemverType = "npm"
	SemverTypeYarn   WorkflowSemverType = "yarn"
	SemverTypeFile   WorkflowSemverType = "file"
	SemverTypePoetry WorkflowSemverType = "poetry"
	SemverTypeDebian WorkflowSemverType = "debian"

	DefaultVersionPattern = "${{%s.version}}-${{cds.run_number}}.sha.g${{git.sha_short}}"
)

type V2Workflow struct {
	Name          string                   `json:"name" jsonschema:"example=my-workflow" jsonschema_description:"Workflow name" jsonschema_extras:"order=1"`
	Repository    *WorkflowRepository      `json:"repository,omitempty" jsonschema_description:"Repository that will be use in the git context ( used by the action for example )" jsonschema_extras:"order=99"`
	OnRaw         json.RawMessage          `json:"on,omitempty" jsonschema_description:"Specify the way to trigger the workflow" jsonschema_extras:"order=2"`
	CommitStatus  *CommitStatus            `json:"commit-status,omitempty"  jsonschema_description:"Specify data send to the build status ( title and description )" jsonschema_extras:"order=3"`
	On            *WorkflowOn              `json:"-" yaml:"-"`
	Stages        map[string]WorkflowStage `json:"stages,omitempty"  jsonschema_description:"Map of stages used in the workflow" jsonschema_extras:"order=5"`
	Gates         map[string]V2JobGate     `json:"gates,omitempty" jsonschema_description:"Map of gates used in the workflow" jsonschema_extras:"order=6"`
	Jobs          map[string]V2Job         `json:"jobs,omitempty" jsonschema:"oneof_required=jobs" jsonschema_description:"Map of jobs used in the workflow" jsonschema_extras:"order=10"`
	Env           map[string]string        `json:"env,omitempty" jsonschema_description:"Environment variables available in all jobs of the workflow" jsonschema_extras:"order=7"`
	Integrations  []string                 `json:"integrations,omitempty" jsonschema_description:"List of integrations available in all jobs of the workflow" jsonschema_extras:"order=9"`
	VariableSets  []string                 `json:"vars,omitempty" jsonschema_description:"List of VariableSets available in all jobs of the workflow" jsonschema_extras:"order=8"`
	Retention     int64                    `json:"retention,omitempty" jsonschema_extras:"order=99" jsonschema_description:"DEPRECATED:: not used anymore, please check project->retention"`
	Annotations   map[string]string        `json:"annotations,omitempty" jsonschema_description:"Map of annotations. They are free text key/value pairs that can be attached to this workflow for storing additional information." jsonschema_extras:"order=11"`
	Semver        *WorkflowSemver          `json:"semver,omitempty" jsonschema_description:"Define semver strategy to automatically bump version on each run" jsonschema_extras:"order=12"`
	Concurrencies []WorkflowConcurrency    `json:"concurrencies,omitempty" jsonschema_description:"Define concurrency groups that can be used on workflow or jobs to limit the number of concurrent executions" jsonschema_extras:"order=13"`
	Concurrency   string                   `json:"concurrency,omitempty" jsonschema:"example=my-concurrency-group" jsonschema_description:"Define a concurrency group to use for the workflow execution" jsonschema_extras:"order=14"`

	// Template fields
	From       string            `json:"from,omitempty" jsonschema:"oneof_required=from,example=my-template" jsonschema_description:"Template name used to create the workflow" jsonschema_extras:"order=15"`
	Parameters map[string]string `json:"parameters,omitempty" jsonschema:"oneof_required=from" jsonschema_description:"Template parameters" jsonschema_extras:"order=16"`
}

type WorkflowSemver struct {
	From        WorkflowSemverType `json:"from" jsonschema_description:"Type of semver to use (git, helm, npm, yarn, file, cargo, poetry, debian)"`
	Path        string             `json:"path" jsonschema_description:"Path to the file that contains the version"`
	ReleaseRefs []string           `json:"release_refs,omitempty" jsonschema_description:"Git references (branches or tags) that will trigger a version bump"`
	Schema      map[string]string  `json:"schema,omitempty" jsonschema_description:"Schema defining how to compute version for each ref"`
}

type WorkfowSemverSchema map[string]string

type V2WorkflowVersion struct {
	ID                 string    `json:"id" db:"id" cli:"id"`
	Version            string    `json:"version" db:"version" cli:"version"`
	ProjectKey         string    `json:"project_key" db:"project_key"`
	WorkflowVCS        string    `json:"workflow_vcs" db:"workflow_vcs"`
	WorkflowRepository string    `json:"workflow_repository" db:"workflow_repository"`
	WorkflowRef        string    `json:"workflow_ref" db:"workflow_ref"`
	WorkflowSha        string    `json:"workflow_sha" db:"workflow_sha"`
	VCSServer          string    `json:"vcs_server" db:"vcs_server" cli:"vcs_server"`
	Repository         string    `json:"repository" db:"repository" cli:"repository"`
	WorkflowName       string    `json:"workflow_name" db:"workflow_name"`
	WorkflowRunID      string    `json:"workflow_run_id" db:"workflow_run_id" cli:"workflow_run_id"`
	Username           string    `json:"username" db:"username" cli:"username"`
	UserID             string    `json:"user_id" db:"user_id"`
	Sha                string    `json:"sha" db:"sha" cli:"sha"`
	Ref                string    `json:"ref" db:"ref" cli:"ref"`
	Type               string    `json:"type" db:"type" cli:"type"`
	File               string    `json:"file" db:"file" cli:"file"`
	Created            time.Time `json:"created" db:"created" cli:"created"`
}

type CommitStatus struct {
	Title       string `json:"title,omitempty" jsonschema:"example=Build ${{cds.version}}" jsonschema_description:"Title sent to the build status on the current commit"`
	Description string `json:"description,omitempty" jsonschema:"example=Build triggered by ${{git.author}}" jsonschema_description:"Description sent to the build status on the current commit"`
}

type WorkflowOn struct {
	Push               *WorkflowOnPush               `json:"push,omitempty" jsonschema_description:"Trigger the workflow on git push event"`
	PullRequest        *WorkflowOnPullRequest        `json:"pull-request,omitempty" jsonschema_description:"Trigger the workflow on pullrequest event"`
	PullRequestComment *WorkflowOnPullRequestComment `json:"pull-request-comment,omitempty" jsonschema_description:"Trigger the workflow on git push event"`
	ModelUpdate        *WorkflowOnModelUpdate        `json:"model-update,omitempty" jsonschema_description:"Trigger the workflow when a worker model is updated (for distant workflow only)"`
	WorkflowUpdate     *WorkflowOnWorkflowUpdate     `json:"workflow-update,omitempty" jsonschema_description:"Trigger the workflow when updated (for distant workflow only)"`
	Schedule           []WorkflowOnSchedule          `json:"schedule,omitempty" jsonschema_description:"Trigger the workflow regarding a cron scheduler"`
	WorkflowRun        []WorkflowOnRun               `json:"workflow-run,omitempty" jsonschema_description:"Trigger the workflow at the end of another workflow run"`
}

type WorkflowOnRun struct {
	Workflow string   `json:"workflow" jsonschema_description:"Name of the workflow to watch"`
	Status   []string `json:"status,omitempty" jsonschema_description:"List of workflow run status to watch"`
	Branches []string `json:"branches,omitempty" jsonschema_description:"Git branches that will trigger the workflow"`
	Tags     []string `json:"tags,omitempty" jsonschema_description:"Git tags that will trigger the workflow"`
}

type WorkflowOnSchedule struct {
	Cron     string `json:"cron" jsonschema:"example=0 */2 * * *" jsonschema_description:"Cron expression defining the schedule"`
	Timezone string `json:"timezone" jsonschema:"example=UTC" jsonschema_description:"Timezone for the cron expression"`
}

type WorkflowOnPush struct {
	Branches []string `json:"branches,omitempty" jsonschema_description:"Git branches that will trigger the workflow"`
	Tags     []string `json:"tags,omitempty" jsonschema_description:"Git tags that will trigger the workflow"`
	Paths    []string `json:"paths,omitempty" jsonschema_description:"File paths that will trigger the workflow when modified"`
	Commit   string   `json:"commit,omitempty" jsonschema_description:"Commit message pattern that will trigger the workflow"`
}

type WorkflowOnPullRequest struct {
	Branches []string                `json:"branches,omitempty" jsonschema_description:"Destination branches that will trigger the workflow"`
	Comment  string                  `json:"comment,omitempty" jsonschema_description:"Comment message pattern that will trigger the workflow"`
	Paths    []string                `json:"paths,omitempty" jsonschema_description:"File paths that will trigger the workflow when modified"`
	Types    []WorkflowHookEventType `json:"types,omitempty" jsonschema_description:"Pull request event types that will trigger the workflow"`
}

type WorkflowOnPullRequestComment struct {
	Branches []string `json:"branches,omitempty" jsonschema_description:"Destination branches that will trigger the workflow"`
	Comment  string   `json:"comment,omitempty"  jsonschema_description:"Comment message pattern that will trigger the workflow"`
	Paths    []string `json:"paths,omitempty" jsonschema_description:"File paths that will trigger the workflow when modified"`
	Types    []string `json:"types,omitempty" jsonschema_description:"Pull request event types that will trigger the workflow"`
}

type WorkflowOnModelUpdate struct {
	Models       []string `json:"models,omitempty" jsonschema_description:"Worker model names that will trigger the workflow"`
	TargetBranch string   `json:"target_branch,omitempty" jsonschema_description:"Git branch that will be used to trigger the workflow"`
}

type WorkflowOnWorkflowUpdate struct {
	TargetBranch string `json:"target_branch,omitempty"  jsonschema_description:"Git branch that will be used to trigger the workflow"`
}

type WorkflowRepository struct {
	VCSServer                   string `json:"vcs,omitempty" jsonschema_extras:"order=1" jsonschema_description:"Server that host the git repository"`
	Name                        string `json:"name,omitempty" jsonschema_extras:"order=2" jsonschema_description:"Name of the git repository: <org>/<name>"`
	InsecureSkipSignatureVerify bool   `json:"insecure_skip_signature_verify,omitempty" jsonschema_extras:"order=3"  jsonschema_description:"Disable the check of signature from the source repository"`
}

func (w V2Workflow) MarshalJSON() ([]byte, error) {
	type Alias V2Workflow // prevent recursion
	workflowAlias := Alias(w)

	// Check default value
	if workflowAlias.On != nil {
		keys := IsDefaultHooks(workflowAlias.On)
		if len(keys) > 0 {
			bts, _ := json.Marshal(keys)
			workflowAlias.OnRaw = bts
		} else {
			onBts, err := json.Marshal(workflowAlias.On)
			if err != nil {
				return nil, WithStack(err)
			}
			workflowAlias.OnRaw = onBts
		}
	}
	bts, err := json.Marshal(workflowAlias)
	return bts, err
}

func IsDefaultHooks(on *WorkflowOn) []WorkflowHookEventName {
	hookKeys := make([]WorkflowHookEventName, 0)
	if on.Push != nil {
		hookKeys = append(hookKeys, WorkflowHookEventNamePush)
		if len(on.Push.Paths) > 0 || len(on.Push.Branches) > 0 || len(on.Push.Tags) > 0 || on.Push.Commit != "" {
			return nil
		}
	}
	if on.PullRequest != nil {
		hookKeys = append(hookKeys, WorkflowHookEventNamePullRequest)
		if len(on.PullRequest.Paths) > 0 || len(on.PullRequest.Branches) > 0 || on.PullRequest.Comment != "" || len(on.PullRequest.Types) > 0 {
			return nil
		}
	}
	if on.PullRequestComment != nil {
		hookKeys = append(hookKeys, WorkflowHookEventNamePullRequestComment)
		if len(on.PullRequestComment.Paths) > 0 || len(on.PullRequestComment.Branches) > 0 || on.PullRequestComment.Comment != "" || len(on.PullRequestComment.Types) > 0 {
			return nil
		}
	}
	if on.WorkflowUpdate != nil {
		hookKeys = append(hookKeys, WorkflowHookEventNameWorkflowUpdate)
		if on.WorkflowUpdate.TargetBranch != "" {
			return nil
		}
	}
	if on.ModelUpdate != nil {
		hookKeys = append(hookKeys, WorkflowHookEventNameModelUpdate)
		if on.ModelUpdate.TargetBranch != "" || len(on.ModelUpdate.Models) > 0 {
			return nil
		}
	}
	if len(on.Schedule) > 0 {
		return nil
	}
	if len(on.WorkflowRun) > 0 {
		return nil
	}
	return hookKeys
}

func (w *V2Workflow) UnmarshalJSON(data []byte) error {
	type Alias V2Workflow // prevent recursion
	var workflowAlias Alias
	if err := JSONUnmarshal(data, &workflowAlias); err != nil {
		return err
	}
	defer func() { *w = V2Workflow(workflowAlias) }()
	if workflowAlias.OnRaw == nil {
		return nil
	}

	bts, _ := json.Marshal(workflowAlias.OnRaw)

	var on WorkflowOn
	if err := JSONUnmarshal(bts, &on); err == nil {
		workflowAlias.On = &on
		return nil
	}

	var onSlice []WorkflowHookEventName
	if err := JSONUnmarshal(bts, &onSlice); err != nil {
		return err
	}
	if len(onSlice) > 0 {
		workflowAlias.On = &WorkflowOn{}
		for _, s := range onSlice {
			switch s {
			case WorkflowHookEventNameWorkflowUpdate:
				workflowAlias.On.WorkflowUpdate = &WorkflowOnWorkflowUpdate{
					TargetBranch: "", // empty for default branch
				}
			case WorkflowHookEventNameModelUpdate:
				workflowAlias.On.ModelUpdate = &WorkflowOnModelUpdate{
					TargetBranch: "",         // empty for default branch
					Models:       []string{}, // empty for all model used on the workflow
				}
			case WorkflowHookEventNamePush:
				workflowAlias.On.Push = &WorkflowOnPush{
					Branches: []string{}, // trigger for all pushed branches
					Paths:    []string{},
					Tags:     []string{},
				}
			case WorkflowHookEventNamePullRequest:
				workflowAlias.On.PullRequest = &WorkflowOnPullRequest{
					Branches: []string{},
					Paths:    []string{},
				}
			case WorkflowHookEventNamePullRequestComment:
				workflowAlias.On.PullRequestComment = &WorkflowOnPullRequestComment{
					Branches: []string{},
					Paths:    []string{},
				}
			}
		}
	}

	return nil
}

type WorkflowStage struct {
	Needs []string `json:"needs,omitempty" jsonschema_description:"Stage dependencies (e.g., [build, test])"`
}

type WorkflowConcurrency struct {
	Name             string           `json:"name" jsonschema:"example=deploy-production" jsonschema_description:"Name of the concurrency rule"`
	Order            ConcurrencyOrder `json:"order,omitempty" jsonschema:"example=oldest_first" jsonschema_description:"Resolving order of the rule: oldest_first | newest_first"`
	Pool             int64            `json:"pool,omitempty" jsonschema:"example=1" jsonschema_description:"Number of concurrent executions allowed for this concurrency rule"`
	CancelInProgress bool             `json:"cancel-in-progress" jsonschema:"example=false" jsonschema_description:"If true, when a new execution is triggered, and oldest in-progress executions are canceled"`
	If               string           `json:"if" jsonschema:"example=${{ git.branch == 'main' }}" jsonschema_description:"Condition to apply the concurrency rule"`
}

type V2Job struct {
	Name            string                  `json:"name,omitempty" jsonschema:"example=my-job" jsonschema_extras:"order=1" jsonschema_description:"Name of the job"`
	If              string                  `json:"if,omitempty" jsonschema:"example=${{ git.branch == 'main' }}" jsonschema_extras:"order=5,textarea=true" jsonschema_description:"Condition to execute the job"`
	Gate            string                  `json:"gate,omitempty" jsonschema_extras:"order=5" jsonschema_description:"Gate allows to trigger manually a job"`
	Steps           []ActionStep            `json:"steps,omitempty" jsonschema:"oneof=steps" jsonschema_extras:"order=11" jsonschema_description:"List of steps"`
	Needs           []string                `json:"needs,omitempty" jsonschema_extras:"order=6,mode=tags" jsonschema_description:"Job dependencies"`
	Stage           string                  `json:"stage,omitempty" jsonschema_extras:"order=2" jsonschema_description:"Stage where the job will be executed"`
	Region          string                  `json:"region,omitempty" jsonschema:"example=eu-west-1" jsonschema_extras:"order=3" jsonschema_description:"Region where the job will be executed"`
	ContinueOnError bool                    `json:"continue-on-error,omitempty" jsonschema:"example=false" jsonschema_extras:"order=4" jsonschema_description:"Allow the workflow to continue even if it fails"`
	RunsOnRaw       json.RawMessage         `json:"runs-on,omitempty" jsonschema:"example=library/default-container" jsonschema_extras:"required,order=5,mode=split" jsonschema_description:"Worker model specification to run the job"`
	RunsOn          V2JobRunsOn             `json:"-"`
	Strategy        *V2JobStrategy          `json:"strategy,omitempty" jsonschema_extras:"order=7" jsonschema_description:"Define a matrix strategy to run multiple times the job with different parameters"`
	Integrations    []string                `json:"integrations,omitempty" jsonschema_extras:"required,order=9" jsonschema_description:"Job integrations"`
	VariableSets    []string                `json:"vars,omitempty" jsonschema_extras:"required,order=10" jsonschema_description:"VariableSet linked to the job"`
	Env             map[string]string       `json:"env,omitempty"  jsonschema_extras:"order=12,mode=edit" jsonschema_description:"Environment variable available in the job"`
	Services        map[string]V2JobService `json:"services,omitempty" jsonschema_description:"Services that must be started and linked to the job"`
	Outputs         map[string]ActionOutput `json:"outputs,omitempty" jsonschema_description:"Outputs exported by the job"`
	From            string                  `json:"from,omitempty" jsonschema:"oneof=from" jsonschema_description:"Job template name used to create the job"`
	Parameters      map[string]string       `json:"parameters,omitempty" jsonschema:"oneof=from" jsonschema_description:"Job template parameters"`
	Concurrency     string                  `json:"concurrency,omitempty" jsonschema_description:"Concurrency rule to apply to the job"`
}

func (j V2Job) Copy() V2Job {
	new := j
	new.Env = make(map[string]string)
	for k, v := range j.Env {
		new.Env[k] = v
	}
	new.Integrations = make([]string, 0, len(j.Integrations))
	new.Integrations = append(new.Integrations, j.Integrations...)

	new.Parameters = make(map[string]string)
	for k, v := range j.Parameters {
		new.Parameters[k] = v
	}
	new.Services = make(map[string]V2JobService)
	for k, v := range j.Services {
		newService := v
		newService.Env = make(map[string]string)
		for envK, envV := range v.Env {
			newService.Env[envK] = envV
		}
		new.Services[k] = newService
	}
	new.VariableSets = make([]string, 0, len(j.VariableSets))
	new.VariableSets = append(new.VariableSets, j.VariableSets...)

	new.Steps = make([]ActionStep, 0, len(j.Steps))
	for _, v := range j.Steps {
		as := v
		as.Env = make(map[string]string)
		for kEnv, vEnv := range v.Env {
			as.Env[kEnv] = vEnv
		}
		as.With = make(map[string]interface{})
		for kWith, vWith := range v.With {
			as.With[kWith] = vWith
		}
		new.Steps = append(new.Steps, as)
	}

	return new
}

type V2JobRunsOn struct {
	Model  string `json:"model" jsonschema_description:"Worker model name to use for the job"`
	Memory string `json:"memory" jsonschema_description:"Amount of memory to use for the job"`
	Flavor string `json:"flavor" jsonschema_description:"Worker flavor to use for the job"`
}

type V2JobGate struct {
	If        string                    `json:"if,omitempty" jsonschema_extras:"order=1,textarea=true" jsonschema_description:"Condition to execute the gate" jsonschema:"example=${{ success() && gate.manual }}"`
	Inputs    map[string]V2JobGateInput `json:"inputs,omitempty" jsonschema_extras:"order=2,mode=edit" jsonschema_description:"Gate inputs to fill for manual triggering"`
	Reviewers V2JobGateReviewers        `json:"reviewers,omitempty" jsonschema_extras:"order=3" jsonschema_description:"Restrict the gate to a list of reviewers"`
}

type V2JobGateInput struct {
	Type        string            `json:"type" jsonschema_description:"Type of the input: boolean | number (Default string)"`
	Default     interface{}       `json:"default,omitempty" jsonschema_description:"Default value of the input"`
	Options     *V2JobGateOptions `json:"options,omitempty"`
	Description string            `json:"description,omitempty" jsonschema_description:"Description of the input"`
}

type V2JobGateOptions struct {
	Multiple bool          `json:"multiple" jsonschema_description:"Allow multiple values selection"`
	Values   []interface{} `json:"values" jsonschema_description:"List of allowed values"`
}

type V2JobGateReviewers struct {
	Groups []string `json:"groups,omitempty" jsonschema_description:"Groups allowed to trigger the gate"`
	Users  []string `json:"users,omitempty" jsonschema_description:"Users allowed to trigger the gate"`
}

func (job *V2Job) Clean() {
	for stepIndex := range job.Steps {
		step := &job.Steps[stepIndex]
		step.Run = CleanString(step.Run)
	}
	for kParam, vParam := range job.Parameters {
		job.Parameters[kParam] = CleanString(vParam)
	}
}

func (job V2Job) Value() (driver.Value, error) {
	j, err := yaml.Marshal(job)
	return j, WrapError(err, "cannot marshal V2Job")
}

func (w *V2Job) Scan(src interface{}) error {
	if src == nil {
		return nil
	}
	source, ok := src.(string)
	if !ok {
		return WithStack(fmt.Errorf("type assertion .(string) failed (%T)", src))
	}
	return WrapError(yaml.Unmarshal([]byte(source), w), "cannot unmarshal V2Job")
}

func (job V2Job) MarshalJSON() ([]byte, error) {
	type Alias V2Job // prevent recursion
	jobAlias := Alias(job)

	if jobAlias.RunsOn.Memory == "" && jobAlias.RunsOn.Flavor == "" {
		runOnsBts, err := json.Marshal(jobAlias.RunsOn.Model)
		if err != nil {
			return nil, WrapError(err, "unable to marshal RunsOn field")
		}
		jobAlias.RunsOnRaw = runOnsBts
	} else {
		runOnsBts, err := json.Marshal(jobAlias.RunsOn)
		if err != nil {
			return nil, WrapError(err, "unable to marshal RunsOn field")
		}
		jobAlias.RunsOnRaw = runOnsBts
	}
	j, err := json.Marshal(jobAlias)
	return j, WrapError(err, "cannot marshal V2Job")
}

func (job *V2Job) UnmarshalJSON(data []byte) error {
	type Alias V2Job // prevent recursion
	var jobAlias Alias
	if err := JSONUnmarshal(data, &jobAlias); err != nil {
		return WrapError(err, "unable to unmarshal v2Job")
	}
	if jobAlias.RunsOnRaw != nil {
		bts, _ := json.Marshal(jobAlias.RunsOnRaw)
		var modelOnly string
		if err := JSONUnmarshal(bts, &modelOnly); err != nil {
			var runsOn V2JobRunsOn
			if err := JSONUnmarshal(bts, &runsOn); err != nil {
				return WrapError(err, "unable to unmarshal RunsOn in V2Job")
			}
			jobAlias.RunsOn = runsOn
		} else {
			runsOn := V2JobRunsOn{
				Model: modelOnly,
			}
			jobAlias.RunsOn = runsOn
		}
	}
	*job = V2Job(jobAlias)
	return nil
}

type V2JobService struct {
	Image     string                `json:"image" jsonschema_extras:"order=1,required" jsonschema_description:"Docker Image"`
	Env       map[string]string     `json:"env,omitempty" jsonschema_extras:"order=2" jsonschema_description:"Environment variables"`
	Readiness V2JobServiceReadiness `json:"readiness,omitempty" jsonschema_extras:"order=3" jsonschema_description:"Service readiness"`
}

type V2JobServiceReadiness struct {
	Command  string `json:"command" jsonschema_extras:"order=1,required" jsonschema_description:"Command executed to check if the service is ready"`
	Interval string `json:"interval" jsonschema_extras:"order=2,required" jsonschema_description:"Internal, example: 10s"`
	Retries  int    `json:"retries" jsonschema_extras:"order=4,required" jsonschema_description:"Nb of retries, example: 5"`
	Timeout  string `json:"timeout" jsonschema_extras:"order=3,required" jsonschema_description:"Timeout, example: 3s"`
}

type V2WorkflowHook struct {
	ID             string             `json:"id" db:"id"`
	ProjectKey     string             `json:"project_key" db:"project_key"`
	VCSName        string             `json:"vcs_name" db:"vcs_name"`
	RepositoryName string             `json:"repository_name" db:"repository_name"`
	EntityID       string             `json:"entity_id" db:"entity_id"`
	WorkflowName   string             `json:"workflow_name" db:"workflow_name"`
	Ref            string             `json:"ref" db:"ref"`
	Commit         string             `json:"commit" db:"commit"`
	Type           string             `json:"type" db:"type"`
	Data           V2WorkflowHookData `json:"data" db:"data"`
	Head           bool               `json:"head" db:"head"`
}

type V2WorkflowHookShort struct {
	ID             string `json:"id" `
	VCSName        string `json:"vcs_name"`
	RepositoryName string `json:"repository_name"`
	WorkflowName   string `json:"workflow_name"`
}

type V2WorkflowScheduleEvent struct {
	Schedule string `json:"schedule"`
}

type V2WorkflowHookData struct {
	VCSServer                   string                  `json:"vcs_server,omitempty"`
	RepositoryName              string                  `json:"repository_name,omitempty"`
	RepositoryEvent             WorkflowHookEventName   `json:"repository_event,omitempty"`
	Model                       string                  `json:"model,omitempty"`
	CommitFilter                string                  `json:"commit_filter,omitempty"`
	BranchFilter                []string                `json:"branch_filter,omitempty"`
	TagFilter                   []string                `json:"tag_filter,omitempty"`
	PathFilter                  []string                `json:"path_filter,omitempty"`
	TypesFilter                 []WorkflowHookEventType `json:"types_filter,omitempty"`
	TargetBranch                string                  `json:"target_branch,omitempty"`
	TargetTag                   string                  `json:"target_tag,omitempty"`
	Cron                        string                  `json:"cron,omitempty"`
	CronTimeZone                string                  `json:"cron_timezone,omitempty"`
	WorkflowRunName             string                  `json:"workflow_run_name"`
	WorkflowRunStatus           []string                `json:"workflow_run_status"`
	InsecureSkipSignatureVerify bool                    `json:"insecure_skip_signature_verify"`
}

func (d V2WorkflowHookData) ValidateRef(ctx context.Context, ref string) bool {
	valid := false

	// If no filter set, hook is ok
	if len(d.BranchFilter) == 0 && len(d.TagFilter) == 0 {
		return true
	}

	if strings.HasPrefix(ref, GitRefBranchPrefix) {
		if len(d.BranchFilter) > 0 || len(d.TagFilter) == 0 {
			valid = IsValidHookRefs(ctx, d.BranchFilter, strings.TrimPrefix(ref, GitRefBranchPrefix))
		}
	} else {
		if len(d.BranchFilter) == 0 || len(d.TagFilter) > 0 {
			valid = IsValidHookRefs(ctx, d.TagFilter, strings.TrimPrefix(ref, GitRefTagPrefix))
		}
	}
	return valid
}

func (w V2WorkflowHookData) Value() (driver.Value, error) {
	j, err := json.Marshal(w)
	return j, WrapError(err, "cannot marshal V2WorkflowHookData")
}

func (w *V2WorkflowHookData) Scan(src interface{}) error {
	if src == nil {
		return nil
	}
	source, ok := src.([]byte)
	if !ok {
		return WithStack(fmt.Errorf("type assertion .([]byte) failed (%T)", src))
	}
	return WrapError(JSONUnmarshal(source, w), "cannot unmarshal V2WorkflowHookData")
}

type V2JobStrategy struct {
	Matrix map[string]interface{} `json:"matrix" jsonschema_description:"Matrix values for the job"`
}

type V2JobConcurrency struct{}

func (w *V2Workflow) Clean() {
	if w.CommitStatus != nil {
		w.CommitStatus.Description = CleanString(w.CommitStatus.Description)
	}
	for k := range w.Jobs {
		job := w.Jobs[k]
		(&job).Clean()
		w.Jobs[k] = job
	}
}

func (w V2Workflow) GetName() string {
	return w.Name
}

func (w V2Workflow) Lint() []error {
	// Before anything, check if workflow inherits from a workflow template.
	// Skip other checks if it is the case.
	if w.From != "" {
		return nil
	}

	errs := w.CheckStageAndJobNeeds()

	if err := w.CheckSemver(); err != nil {
		errs = append(errs, err)
	}

	errGates := w.CheckGates()
	if len(errGates) > 0 {
		errs = append(errs, errGates...)
	}

	workflowSchema := GetWorkflowJsonSchema(nil, nil, nil)
	workflowSchemaS, err := workflowSchema.MarshalJSON()
	if err != nil {
		return []error{NewErrorFrom(err, "workfow %s: unable to load workflow schema", w.Name)}
	}
	schemaLoader := gojsonschema.NewStringLoader(string(workflowSchemaS))

	modelJson, err := json.Marshal(w)
	if err != nil {
		return []error{NewErrorFrom(err, "workfow %s: unable to marshal workflow", w.Name)}
	}
	documentLoader := gojsonschema.NewStringLoader(string(modelJson))

	if w.On != nil {
		for _, s := range w.On.Schedule {
			if _, err := cronexpr.Parse(s.Cron); err != nil {
				errs = append(errs, NewErrorFrom(err, "workflow %s: unable to parse cron expression: %s", w.Name, s.Cron))
			}
		}
	}

	result, err := gojsonschema.Validate(schemaLoader, documentLoader)
	if err != nil {
		return []error{NewErrorFrom(ErrInvalidData, "workflow %s: unable to validate file: %v", w.Name, err.Error())}
	}

	for _, e := range result.Errors() {
		errs = append(errs, NewErrorFrom(ErrInvalidData, "workflow %s: yaml validation failed: %s", w.Name, e.String()))
	}

	if len(errs) > 0 {
		return errs
	}
	return nil
}

func (w V2Workflow) CheckGates() []error {
	errs := make([]error, 0)
	for jobID, j := range w.Jobs {
		if j.Gate != "" {
			if _, has := w.Gates[j.Gate]; !has {
				errs = append(errs, NewErrorFrom(ErrInvalidData, "workflow %s job %s: gate %s not found", w.Name, jobID, j.Gate))
			}
		}
	}

	for gateName, g := range w.Gates {
		if g.If == "" {
			errs = append(errs, NewErrorFrom(ErrInvalidData, "workflow %s gate %s: if cannot be empty", w.Name, gateName))
		}
		for k, gateInput := range g.Inputs {
			if gateInput.Options != nil && gateInput.Options.Multiple && gateInput.Default != nil {
				if _, ok := gateInput.Default.([]interface{}); !ok {
					errs = append(errs, NewErrorFrom(ErrInvalidData, "workflow %s gate %s input %s: default value must be an array", w.Name, gateName, k))
				}
			}
		}
	}
	return errs
}

func (w V2Workflow) CheckSemver() error {
	if w.Semver == nil {
		return nil
	}
	found := false
	for _, a := range AvailableSemverType {
		if a == w.Semver.From {
			found = true
			break
		}
	}
	if !found {
		return NewErrorFrom(ErrInvalidData, "workflow %s: semver from %s not implemented", w.Name, w.Semver.From)
	}

	if w.Semver.From == SemverTypeGit && w.Semver.Path != "" {
		return NewErrorFrom(ErrInvalidData, "workflow %s: semver.path is not allowed for semver from git", w.Name)
	}
	if w.Semver.From != SemverTypeGit && w.Semver.Path == "" {
		return NewErrorFrom(ErrInvalidData, "workflow %s: missing required field semver.path", w.Name)
	}
	if w.Semver.From == SemverTypeGit && len(w.Semver.ReleaseRefs) > 0 {
		return NewErrorFrom(ErrInvalidData, " workflow %s: semver.release_refs is not allowed for semver from git", w.Name)
	}
	return nil
}

func (w V2Workflow) CheckStageAndJobNeeds() []error {
	errs := make([]error, 0)
	if len(w.Stages) > 0 {
		stages := make(map[string]WorkflowStage)
		jobs := make(map[string]V2Job)
		for k, v := range w.Stages {
			stages[k] = v
		}
		for k, v := range w.Jobs {
			jobs[k] = v
		}
		// Check stage needs
		for k := range stages {
			for _, n := range stages[k].Needs {
				if _, exist := stages[n]; !exist {
					errs = append(errs, NewErrorFrom(ErrInvalidData, "workflow %s stage %s: needs not found %s", w.Name, k, n))
				}
			}
		}
		// Check job needs
		for k, j := range w.Jobs {
			if j.Stage == "" {
				errs = append(errs, NewErrorFrom(ErrInvalidData, "workflow %s: missing stage on job %s", w.Name, k))
				continue
			}
			if _, stageExist := stages[j.Stage]; !stageExist {
				errs = append(errs, NewErrorFrom(ErrInvalidData, "workflow %s stage %s on job %s does not exist", w.Name, j.Stage, k))
			}
			for _, n := range j.Needs {
				jobNeed, exist := jobs[n]
				if !exist {
					errs = append(errs, NewErrorFrom(ErrInvalidData, "workflow %s job %s: needs not found %s", w.Name, k, n))
				}
				if jobNeed.Stage != j.Stage {
					errs = append(errs, NewErrorFrom(ErrInvalidData, "workflow %s job %s: need %s must be in the same stage", w.Name, k, n))
				}
				if n == k {
					errs = append(errs, NewErrorFrom(ErrInvalidData, "workflow %s job %s: a job cannot depend of itself", w.Name, k))
				}
			}
		}
	} else {
		for k, j := range w.Jobs {
			if j.Stage != "" {
				errs = append(errs, NewErrorFrom(ErrInvalidData, "workflow %s stage %s on job %s does not exist", w.Name, j.Stage, k))
			}
			for _, n := range j.Needs {
				if _, exist := w.Jobs[n]; !exist {
					errs = append(errs, NewErrorFrom(ErrInvalidData, "workflow %s job %s: needs not found [%s]", w.Name, k, n))
				}
				if n == k {
					errs = append(errs, NewErrorFrom(ErrInvalidData, "workflow %s job %s: a job cannot depend of itself", w.Name, k))
				}
			}
		}
	}
	if len(errs) > 0 {
		return errs
	}
	return nil
}

func WorkflowJobParents(w V2Workflow, jobID string) []string {
	needsParents := WorkflowJobParentsNeeds(w, jobID)
	if len(w.Stages) == 0 {
		return needsParents
	}

	currentStage := w.Jobs[jobID].Stage
	parentStages := WorkflowStageParentsNeeds(w, currentStage)

	for jobID, j := range w.Jobs {
		if slices.Contains(parentStages, j.Stage) {
			needsParents = append(needsParents, jobID)
		}
	}

	return needsParents
}

func WorkflowStageParentsNeeds(w V2Workflow, currentStage string) []string {
	parents := make([]string, 0)
	stage := w.Stages[currentStage]
	for _, n := range stage.Needs {
		needParents := WorkflowStageParentsNeeds(w, n)
		parents = append(parents, needParents...)
		parents = append(parents, n)
	}
	return parents
}

func WorkflowJobParentsNeeds(w V2Workflow, jobID string) []string {
	parents := make([]string, 0)
	currentJob := w.Jobs[jobID]
	for _, n := range currentJob.Needs {
		needParents := WorkflowJobParentsNeeds(w, n)
		parents = append(parents, needParents...)
		parents = append(parents, n)
	}
	return parents
}

type V2WorkflowRunManualRequest struct {
	Branch           string `json:"branch,omitempty"`
	Tag              string `json:"tag,omitempty"`
	Sha              string `json:"sha,omitempty"`
	WorkflowBranch   string `json:"workflow_branch,omitempty"`
	WorkflowTag      string `json:"workflow_tag,omitempty"`
	TargetRepository string `json:"target_repository,omitempty`
}

type V2WorkflowRunManualResponse struct {
	HookEventUUID string `json:"hook_event_uuid"`
	UIUrl         string `json:"ui_url"`
}

type SchedulerExecution struct {
	SchedulerDef      V2WorkflowHook
	NextExecutionTime int64
}
