package sdk

import (
	"encoding/json"
	"github.com/xeipuuv/gojsonschema"
)

type WorkerModelTemplate struct {
	Name string          `json:"name" cli:"name" jsonschema:"required,minLength=1"`
	Type string          `json:"type" cli:"type" jsonschema:"required,enum=docker,enum=vm"`
	Spec json.RawMessage `json:"spec" jsonschema:"required" jsonschema_allof_type:"type=docker:#/$defs/WorkerModelTemplateDocker,type=vm:#/$defs/WorkerModelTemplateVM"`
}

type WorkerModelTemplateDocker struct {
	Cmd   string            `json:"cmd" jsonschema:"required,minLength=1"`
	Shell string            `json:"shell" jsonschema:"required,minLength=1"`
	Envs  map[string]string `json:"envs,omitempty"`
}

type WorkerModelTemplateVM struct {
	Cmd     string `json:"cmd" jsonschema:"required,minLength=1"`
	PreCmd  string `json:"pre_cmd,omitempty"`
	PostCmd string `json:"post_cmd" jsonschema:"required,minLength=1"`
}

func (wmt WorkerModelTemplate) Lint() []error {
	workerModelTemplateSchema := GetWorkerModelTemplateJsonSchema()
	workerModelTemplateSchemaS, err := workerModelTemplateSchema.MarshalJSON()
	if err != nil {
		return []error{NewErrorFrom(err, "unable to load worker model schema")}
	}
	schemaLoader := gojsonschema.NewStringLoader(string(workerModelTemplateSchemaS))

	modelJson, err := json.Marshal(wmt)
	if err != nil {
		return []error{NewErrorFrom(err, "unable to marshal worker model")}
	}
	documentLoader := gojsonschema.NewStringLoader(string(modelJson))

	result, err := gojsonschema.Validate(schemaLoader, documentLoader)
	if err != nil {
		return []error{NewErrorFrom(err, "unable to validate worker model")}
	}
	if result.Valid() {
		return nil
	}

	errors := make([]error, 0, len(result.Errors()))
	for _, e := range result.Errors() {
		errors = append(errors, NewErrorFrom(ErrInvalidData, e.String()))
	}
	return errors
}

func (wmt WorkerModelTemplate) GetName() string {
	return wmt.Name
}
