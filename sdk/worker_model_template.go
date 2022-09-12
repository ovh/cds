package sdk

import (
	"encoding/json"
	"fmt"

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
	multipleError := MultiError{}

	workerModelTemplateSchema := GetWorkerModelTemplateJsonSchema()
	workerModelTemplateSchemaS, err := workerModelTemplateSchema.MarshalJSON()
	if err != nil {
		multipleError.Append(WrapError(err, "unable to load worker model template schema"))
		return multipleError
	}
	schemaLoader := gojsonschema.NewStringLoader(string(workerModelTemplateSchemaS))

	modelJson, err := json.Marshal(wmt)
	if err != nil {
		multipleError.Append(WithStack(err))
		return multipleError
	}
	documentLoader := gojsonschema.NewStringLoader(string(modelJson))

	result, err := gojsonschema.Validate(schemaLoader, documentLoader)
	if err != nil {
		multipleError.Append(WithStack(err))
		return multipleError
	}
	if result.Valid() {
		return nil
	}
	for _, e := range result.Errors() {
		multipleError.Append(fmt.Errorf("%v", e))
	}
	return multipleError
}

func (wmt WorkerModelTemplate) GetName() string {
	return wmt.Name
}
