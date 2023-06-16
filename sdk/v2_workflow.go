package sdk

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"github.com/rockbears/yaml"
	"github.com/xeipuuv/gojsonschema"
)

type V2Workflow struct {
	Name       string                   `json:"name"`
	Repository WorkflowRepository       `json:"repository,omitempty"`
	Stages     map[string]WorkflowStage `json:"stages,omitempty"`
	Jobs       map[string]V2Job         `json:"jobs"`
}

type WorkflowRepository struct {
	VCSServer string `json:"vcs,omitempty" jsonschema_extras:"order=1" jsonschema_description:"Server that host the git repository"`
	Name      string `json:"name,omitempty" jsonschema_extras:"order=2" jsonschema_description:"Name of the git repository: <org>/<name>"`
}

type WorkflowStage struct {
	Needs []string `json:"needs,omitempty" jsonschema_description:"Stage dependencies"`
}

type V2Job struct {
	Name        string            `json:"name" jsonschema_extras:"order=1" jsonschema_description:"Name of the job"`
	If          string            `json:"if,omitempty" jsonschema_extras:"order=2" jsonschema_description:"Condition to execute the job"`
	Inputs      map[string]string `json:"inputs,omitempty" jsonschema_extras:"order=3" jsonschema_description:"Input of thejob"`
	Steps       []ActionStep      `json:"steps,omitempty" jsonschema_extras:"order=5" jsonschema_description:"List of steps"`
	Needs       []string          `json:"needs,omitempty" jsonschema_extras:"order=6" jsonschema_description:"Job dependencies"`
	Stage       string            `json:"stage,omitempty" jsonschema_extras:"order=7"`
	Region      string            `json:"stage,omitempty" jsonschema_extras:"order=8"`
	WorkerModel string            `json:"worker_model,omitempty" jsonschema_extras:"order=9"`
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

	workflowSchema := GetWorkflowJsonSchema(nil)
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
	} else {
		for k, j := range w.Jobs {
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
