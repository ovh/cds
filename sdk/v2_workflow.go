package sdk

import (
	"encoding/json"
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
	Name   string            `json:"name" jsonschema_extras:"order=1" jsonschema_description:"Name of the job"`
	Stage  string            `json:"stage,omitempty" jsonschema_extras:"order=2"`
	If     string            `json:"if,omitempty" jsonschema_extras:"order=3" jsonschema_description:"Condition to execute the job"`
	Inputs map[string]string `json:"inputs,omitempty" jsonschema_extras:"order=4,mode=edit" jsonschema_description:"Input of the job. If you define inputs, your job will only be able to use these inputs and no others variables/contexts."`
	Steps  []ActionStep      `json:"steps,omitempty" jsonschema_extras:"order=5" jsonschema_description:"List of steps"`
	Needs  []string          `json:"needs,omitempty" jsonschema_extras:"order=6" jsonschema_description:"Job dependencies"`

	// TODO
	Concurrency V2JobConcurrency `json:"-"`
	Strategy    V2JobStrategy    `json:"-"`
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
