package sdk

import (
	"fmt"

	"database/sql/driver"
	"encoding/json"
	"github.com/rockbears/yaml"
	"github.com/xeipuuv/gojsonschema"
)

const (
	WorkflowHookTypeRepository  = "RepositoryWebHook"
	WorkflowHookTypeWorkerModel = "WorkerModel"
	WorkflowHookTypeWorkflow    = "Workflow"
)

type V2Workflow struct {
	Name       string                   `json:"name"`
	Repository *WorkflowRepository      `json:"repository,omitempty"`
	On         *WorkflowOn              `json:"on,omitempty"`
	Stages     map[string]WorkflowStage `json:"stages,omitempty"`
	Jobs       map[string]V2Job         `json:"jobs"`
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

type WorkflowStage struct {
	Needs []string `json:"needs,omitempty" jsonschema_description:"Stage dependencies"`
}

type V2Job struct {
	Name        string            `json:"name" jsonschema_extras:"order=1,required" jsonschema_description:"Name of the job"`
	If          string            `json:"if,omitempty" jsonschema_extras:"order=5,textarea=true" jsonschema_description:"Condition to execute the job"`
	Inputs      map[string]string `json:"inputs,omitempty" jsonschema_extras:"order=7,mode=edit" jsonschema_description:"Input of thejob"`
	Steps       []ActionStep      `json:"steps,omitempty" jsonschema_extras:"order=8" jsonschema_description:"List of steps"`
	Needs       []string          `json:"needs,omitempty" jsonschema_extras:"order=6,mode=tags" jsonschema_description:"Job dependencies"`
	Stage       string            `json:"stage,omitempty" jsonschema_extras:"order=2"`
	Region      string            `json:"region,omitempty" jsonschema_extras:"order=3"`
	WorkerModel string            `json:"worker_model,omitempty" jsonschema_extras:"required,order=4,mode=split"`

	// TODO
	Concurrency V2JobConcurrency `json:"-"`
	Strategy    V2JobStrategy    `json:"-"`
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
		return WithStack(fmt.Errorf("type assertion .([]byte) failed (%T)", src))
	}
	return WrapError(yaml.Unmarshal([]byte(source), w), "cannot unmarshal V2Job")
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
}

type V2JobConcurrency struct {
}

func (w V2Workflow) GetName() string {
	return w.Name
}

func (w V2Workflow) Lint() []error {
	errs := make([]error, 0)

	errs = w.CheckStageAndJobNeeds()

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
		for k, v := range w.Stages {
			stages[k] = v
		}
		for k := range stages {
			for _, n := range stages[k].Needs {
				if _, exist := stages[n]; !exist {
					errs = append(errs, NewErrorFrom(ErrInvalidData, "Stage %s: needs not found %s", k, n))
				}
			}
		}
		for k, j := range w.Jobs {
			if len(j.Needs) > 0 {
				errs = append(errs, NewErrorFrom(ErrInvalidData, "As you use stages, you can't add `needs` attribute on job %s", k))
			}
			if j.Stage == "" {
				errs = append(errs, NewErrorFrom(ErrInvalidData, "Missing stage on job %s", k))
			}
			if _, stageExist := stages[j.Stage]; !stageExist {
				errs = append(errs, NewErrorFrom(ErrInvalidData, "Stage %s on job %s does not exist", j.Stage, k))
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
