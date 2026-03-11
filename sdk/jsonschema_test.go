package sdk_test

import (
	"encoding/json"
	"testing"

	"github.com/ovh/cds/sdk"
	"github.com/stretchr/testify/assert"
	"github.com/xeipuuv/gojsonschema"
)

func TestGetYAMLKeywordsFromJsonSchema(t *testing.T) {
	got := sdk.GetYAMLKeywordsFromJsonSchema()

	// DIsplay the obtained keywords
	for _, v := range got {
		t.Log(v)
	}
}

func validateWorkflowSchema(t *testing.T, w sdk.V2Workflow) []string {
	t.Helper()
	workflowSchema := sdk.GetWorkflowJsonSchema(nil, nil, nil)
	workflowSchemaS, err := workflowSchema.MarshalJSON()
	assert.NoError(t, err)
	schemaLoader := gojsonschema.NewStringLoader(string(workflowSchemaS))

	modelJson, err := json.Marshal(w)
	assert.NoError(t, err)
	documentLoader := gojsonschema.NewStringLoader(string(modelJson))

	result, err := gojsonschema.Validate(schemaLoader, documentLoader)
	assert.NoError(t, err)

	var errs []string
	for _, e := range result.Errors() {
		errs = append(errs, e.String())
	}
	return errs
}

func TestWorkflowJsonSchema_StepIDWithHyphen_JobWithoutHyphen(t *testing.T) {
	w := sdk.V2Workflow{
		Name: "test-workflow",
		Jobs: map[string]sdk.V2Job{
			"myJob": {
				RunsOnRaw: json.RawMessage(`"my-model"`),
				Steps: []sdk.ActionStep{
					{ID: "invalid-step-id", Run: "echo hello"},
				},
			},
		},
	}
	errs := validateWorkflowSchema(t, w)
	assert.NotEmpty(t, errs, "step.id with hyphen should be rejected when job name has no hyphen")
}

func TestWorkflowJsonSchema_StepIDWithHyphen_JobWithHyphen(t *testing.T) {
	w := sdk.V2Workflow{
		Name: "test-workflow",
		Jobs: map[string]sdk.V2Job{
			"my-job": {
				RunsOnRaw: json.RawMessage(`"my-model"`),
				Steps: []sdk.ActionStep{
					{ID: "invalid-step-id", Run: "echo hello"},
				},
			},
		},
	}
	errs := validateWorkflowSchema(t, w)
	assert.NotEmpty(t, errs, "step.id with hyphen should be rejected even when job name has a hyphen")
}

func TestWorkflowJsonSchema_StepIDAlphanumeric_JobWithHyphen(t *testing.T) {
	w := sdk.V2Workflow{
		Name: "test-workflow",
		Jobs: map[string]sdk.V2Job{
			"my-job": {
				RunsOnRaw: json.RawMessage(`"my-model"`),
				Steps: []sdk.ActionStep{
					{ID: "validStepId", Run: "echo hello"},
				},
			},
		},
	}
	errs := validateWorkflowSchema(t, w)
	assert.Empty(t, errs, "alphanumeric step.id should be accepted with hyphenated job name")
}

func TestWorkflowJsonSchema_StepIDAlphanumeric_JobWithoutHyphen(t *testing.T) {
	w := sdk.V2Workflow{
		Name: "test-workflow",
		Jobs: map[string]sdk.V2Job{
			"myJob": {
				RunsOnRaw: json.RawMessage(`"my-model"`),
				Steps: []sdk.ActionStep{
					{ID: "validStepId", Run: "echo hello"},
				},
			},
		},
	}
	errs := validateWorkflowSchema(t, w)
	assert.Empty(t, errs, "alphanumeric step.id should be accepted with alphanumeric job name")
}
