package sdk

import (
	"encoding/json"

	"github.com/xeipuuv/gojsonschema"
)

const (
	EntityActionInputKey = "^[a-zA-Z]{1,}$"
	EntityActionStepID   = "^[a-zA-Z]{0,}$"
)

type V2Action struct {
	Name        string                  `json:"name" jsonschema_extras:"order=1" jsonschema_description:"Name of the action"`
	Description string                  `json:"description,omitempty" jsonschema_extras:"order=2"`
	Inputs      map[string]ActionInput  `json:"inputs,omitempty" jsonschema_extras:"order=3,mode=edit" jsonschema_description:"Inputs of the action"`
	Outputs     map[string]ActionOutput `json:"outputs,omitempty" jsonschema_extras:"order=4,mode=edit" jsonschema_description:"Outputs compute by the action"`
	Runs        ActionRuns              `json:"runs" jsonschema_extras:"order=5"`
}

type ActionRuns struct {
	Steps []ActionStep `json:"steps" jsonschema_description:"List of sequential steps executed by the action"`
}

type ActionInput struct {
	Description string `json:"description,omitempty" jsonschema_extras:"order=2"`
	Default     string `json:"default,omitempty" jsonschema_extras:"order=1" jsonschema_description:"Default input value used if the caller do not specified anything"`
}

type ActionOutput struct {
	Description string `json:"description,omitempty" jsonschema_extras:"order=2"`
	Value       string `json:"value" jsonschema_extras:"order=1"`
}

type ActionStep struct {
	ID   string            `json:"id,omitempty" jsonschema_extras:"order=2" jsonschema_description:"Identifier of the step"`
	Uses string            `json:"uses,omitempty" jsonschema:"oneof_required=uses" jsonschema_extras:"order=1,onchange=loadentity,prefix=actions/" jsonschema_description:"Sub action to call"`
	Run  string            `json:"run,omitempty" jsonschema:"oneof_required=run" jsonschema_extras:"order=1,code=true" jsonschema_description:"Script to execute"`
	With map[string]string `json:"with,omitempty" jsonschema:"oneof_not_required=run" jsonschema_extras:"order=3,mode=use" jsonschema_description:"Action parameters"`
}

type ActionStepUsesWith map[string]string

func (a V2Action) GetName() string {
	return a.Name
}

func (a V2Action) Lint() []error {
	actionSchema := GetActionJsonSchema(nil)
	actionSchemaS, err := actionSchema.MarshalJSON()
	if err != nil {
		return []error{NewErrorFrom(err, "unable to load action schema")}
	}
	schemaLoader := gojsonschema.NewStringLoader(string(actionSchemaS))

	modelJson, err := json.Marshal(a)
	if err != nil {
		return []error{NewErrorFrom(err, "unable to marshal action")}
	}
	documentLoader := gojsonschema.NewStringLoader(string(modelJson))

	result, err := gojsonschema.Validate(schemaLoader, documentLoader)
	if err != nil {
		return []error{NewErrorFrom(err, "unable to validate action")}
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
