package sdk

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"

	"github.com/rockbears/yaml"
	"github.com/xeipuuv/gojsonschema"
)

const (
	WorkflowHookTypeRepository  = "RepositoryWebHook"
	WorkflowHookTypeWorkerModel = "WorkerModelUpdate"
	WorkflowHookTypeWorkflow    = "WorkflowUpdate"
)

type V2Workflow struct {
	Name       string                   `json:"name"`
	Repository *WorkflowRepository      `json:"repository,omitempty"`
	OnRaw      json.RawMessage          `json:"on,omitempty"`
	On         *WorkflowOn              `json:"-" yaml:"-"`
	Stages     map[string]WorkflowStage `json:"stages,omitempty"`
	Jobs       map[string]V2Job         `json:"jobs"`
	Env        map[string]string        `json:"env,omitempty"`
}

type WorkflowOn struct {
	Push           *WorkflowOnPush           `json:"push,omitempty"`
	ModelUpdate    *WorkflowOnModelUpdate    `json:"model_update,omitempty"`
	WorkflowUpdate *WorkflowOnWorkflowUpdate `json:"workflow_update,omitempty"`
}

type WorkflowOnPush struct {
	Branches []string `json:"branches,omitempty"`
	Paths    []string `json:"paths,omitempty"`
}

type WorkflowOnModelUpdate struct {
	Models       []string `json:"models,omitempty"`
	TargetBranch string   `json:"target_branch,omitempty"`
}

type WorkflowOnWorkflowUpdate struct {
	TargetBranch string `json:"target_branch,omitempty"`
}

type WorkflowRepository struct {
	VCSServer string `json:"vcs,omitempty" jsonschema_extras:"order=1" jsonschema_description:"Server that host the git repository"`
	Name      string `json:"name,omitempty" jsonschema_extras:"order=2" jsonschema_description:"Name of the git repository: <org>/<name>"`
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

func IsDefaultHooks(on *WorkflowOn) []string {
	hookKeys := make([]string, 0)
	if on.Push != nil {
		hookKeys = append(hookKeys, WorkflowHookEventPush)
		if len(on.Push.Paths) > 0 || len(on.Push.Branches) > 0 {
			return nil
		}
	}
	if on.WorkflowUpdate != nil {
		hookKeys = append(hookKeys, WorkflowHookEventWorkflowUpdate)
		if on.WorkflowUpdate.TargetBranch != "" {
			return nil
		}
	}
	if on.ModelUpdate != nil {
		hookKeys = append(hookKeys, WorkflowHookEventModelUpdate)
		if on.ModelUpdate.TargetBranch != "" || len(on.ModelUpdate.Models) > 0 {
			return nil
		}
	}
	return hookKeys
}

func (w *V2Workflow) UnmarshalJSON(data []byte) error {
	type Alias V2Workflow // prevent recursion
	var workflowAlias Alias
	if err := JSONUnmarshal(data, &workflowAlias); err != nil {
		return WrapError(err, "unable to unmarshal workflow")
	}
	if workflowAlias.OnRaw != nil {
		bts, _ := json.Marshal(workflowAlias.OnRaw)
		var on WorkflowOn
		if err := JSONUnmarshal(bts, &on); err != nil {
			var onSlice []string
			if err := JSONUnmarshal(bts, &onSlice); err != nil {
				return WrapError(err, "unable to unmarshal On in Workflow")
			}
			if len(onSlice) > 0 {
				workflowAlias.On = &WorkflowOn{}
				for _, s := range onSlice {
					switch s {
					case WorkflowHookEventWorkflowUpdate:
						workflowAlias.On.WorkflowUpdate = &WorkflowOnWorkflowUpdate{
							TargetBranch: "", // empty for default branch
						}
					case WorkflowHookEventModelUpdate:
						workflowAlias.On.ModelUpdate = &WorkflowOnModelUpdate{
							TargetBranch: "",         // empty for default branch
							Models:       []string{}, // empty for all model used on the workflow
						}
					case WorkflowHookEventPush:
						workflowAlias.On.Push = &WorkflowOnPush{
							Branches: []string{}, // trigger for all pushed branches
							Paths:    []string{},
						}
					}
				}
			}
		} else {
			workflowAlias.On = &on
		}
	}
	*w = V2Workflow(workflowAlias)
	return nil
}

type WorkflowStage struct {
	Needs []string `json:"needs,omitempty" jsonschema_description:"Stage dependencies"`
}

type V2Job struct {
	Name            string                  `json:"name" jsonschema_extras:"order=1,required" jsonschema_description:"Name of the job"`
	If              string                  `json:"if,omitempty" jsonschema_extras:"order=5,textarea=true" jsonschema_description:"Condition to execute the job"`
	Inputs          map[string]string       `json:"inputs,omitempty" jsonschema_extras:"order=8,mode=edit" jsonschema_description:"Input of thejob"`
	Steps           []ActionStep            `json:"steps,omitempty" jsonschema_extras:"order=10" jsonschema_description:"List of steps"`
	Needs           []string                `json:"needs,omitempty" jsonschema_extras:"order=6,mode=tags" jsonschema_description:"Job dependencies"`
	Stage           string                  `json:"stage,omitempty" jsonschema_extras:"order=2"`
	Region          string                  `json:"region,omitempty" jsonschema_extras:"order=3"`
	ContinueOnError bool                    `json:"continue-on-error,omitempty" jsonschema_extras:"order=4"`
	RunsOn          string                  `json:"runs-on,omitempty" jsonschema_extras:"required,order=5,mode=split"`
	Strategy        *V2JobStrategy          `json:"strategy,omitempty" jsonschema_extras:"order=7"`
	Integrations    *V2JobIntegrations      `json:"integrations,omitempty" jsonschema_extras:"required,order=9" jsonschema_description:"Job integrations"`
	Env             map[string]string       `json:"env,omitempty"  jsonschema_extras:"order=11,mode=edit" jsonschema_description:"Environment variable available in the job"`
	Services        map[string]V2JobService `json:"services,omitempty"`

	// TODO
	Concurrency V2JobConcurrency `json:"-"`
}

func (w V2Job) Value() (driver.Value, error) {
	j, err := yaml.Marshal(w)
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

type V2JobIntegrations struct {
	Artifacts  string `json:"artifacts,omitempty"`
	Deployment string `json:"deployment,omitempty"`
}

func (v *V2JobIntegrations) IsEmpty() bool {
	if v == nil {
		return true
	}
	return v.Artifacts == "" && v.Deployment == ""
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
	Branch         string             `json:"branch" db:"branch"`
	Commit         string             `json:"commit" db:"commit"`
	Type           string             `json:"type" db:"type"`
	Data           V2WorkflowHookData `json:"data" db:"data"`
}

type V2WorkflowHookData struct {
	VCSServer       string   `json:"vcs_server,omitempty"`
	RepositoryName  string   `json:"repository_name,omitempty"`
	RepositoryEvent string   `json:"repository_event,omitempty"`
	Model           string   `json:"model,omitempty"`
	BranchFilter    []string `json:"branch_filter,omitempty"`
	PathFilter      []string `json:"path_filter,omitempty"`
	TargetBranch    string   `json:"target_branch,omitempty"`
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
	Matrix map[string][]string `json:"matrix"`
}

type V2JobConcurrency struct {
}

func (w V2Workflow) GetName() string {
	return w.Name
}

func (w V2Workflow) Lint() []error {
	errs := w.CheckStageAndJobNeeds()

	workflowSchema := GetWorkflowJsonSchema(nil, nil, nil)
	workflowSchemaS, err := workflowSchema.MarshalJSON()
	if err != nil {
		return []error{NewErrorFrom(err, "unable to load action schema")}
	}
	schemaLoader := gojsonschema.NewStringLoader(string(workflowSchemaS))

	modelJson, err := json.Marshal(w)
	if err != nil {
		return []error{NewErrorFrom(err, "unable to marshal action")}
	}
	documentLoader := gojsonschema.NewStringLoader(string(modelJson))

	result, err := gojsonschema.Validate(schemaLoader, documentLoader)
	if err != nil {
		return []error{NewErrorFrom(err, "unable to validate action")}
	}

	for _, e := range result.Errors() {
		errs = append(errs, NewErrorFrom(ErrInvalidData, "yaml validation failed: "+e.String()))
	}

	if len(errs) > 0 {
		return errs
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
					errs = append(errs, NewErrorFrom(ErrInvalidData, "Stage %s: needs not found %s", k, n))
				}
			}
		}
		// Check job needs
		for k, j := range w.Jobs {
			if j.Stage == "" {
				errs = append(errs, NewErrorFrom(ErrInvalidData, "Missing stage on job %s", k))
			}
			if _, stageExist := stages[j.Stage]; !stageExist {
				errs = append(errs, NewErrorFrom(ErrInvalidData, "Stage %s on job %s does not exist", j.Stage, k))
			}
			for _, n := range j.Needs {
				jobNeed, exist := jobs[n]
				if !exist {
					errs = append(errs, NewErrorFrom(ErrInvalidData, "Job %s: needs not found %s", k, n))
				}
				if jobNeed.Stage != j.Stage {
					errs = append(errs, NewErrorFrom(ErrInvalidData, "Job %s: need %s must be in the same stage", k, n))
				}
			}
		}
	} else {
		for k, j := range w.Jobs {
			if j.Stage != "" {
				errs = append(errs, NewErrorFrom(ErrInvalidData, "Stage %s on job %s does not exist", j.Stage, k))
			}
			for _, n := range j.Needs {
				if _, exist := w.Jobs[n]; !exist {
					errs = append(errs, NewErrorFrom(ErrInvalidData, "Job %s: needs not found [%s]", k, n))
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
	parents := make([]string, 0)
	currentJob := w.Jobs[jobID]
	for _, n := range currentJob.Needs {
		needParents := WorkflowJobParents(w, n)
		parents = append(parents, needParents...)
		parents = append(parents, n)
	}
	return parents
}
