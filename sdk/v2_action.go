package sdk

import (
	"encoding/json"

	"github.com/xeipuuv/gojsonschema"
)

type V2Action struct {
	Name        string                  `json:"name"`
	Author      string                  `json:"author"`
	Description string                  `json:"description"`
	Inputs      map[string]ActionInput  `json:"inputs"`
	Outputs     map[string]ActionOutput `json:"outputs"`
	Runs        ActionRuns              `json:"runs"`
}

type ActionRuns struct {
	Steps []ActionStep `json:"steps"`
}

type ActionInput struct {
	Description string `json:"description"`
	Required    bool   `json:"required"`
	Default     string `json:"default"`
}

type ActionOutput struct {
	Description string `json:"description"`
	Value       string `json:"value"`
}

type ActionStep struct {
	ID        string `json:"id"`
	Uses      string `json:"uses"`
	Run       string `json:"run"`
	Condition string `json:"if"`
}

func (a V2Action) GetName() string {
	return a.Name
}

func (a V2Action) Lint() []error {
	actionSchema := GetActionJsonSchema()
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
