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
					{ID: "my-step-with-hyphen", Run: "echo hello"},
				},
			},
		},
	}
	errs := validateWorkflowSchema(t, w)
	assert.Empty(t, errs, "step.id with hyphen should be accepted")
}

func TestWorkflowJsonSchema_StepIDWithHyphen_JobWithHyphen(t *testing.T) {
	w := sdk.V2Workflow{
		Name: "test-workflow",
		Jobs: map[string]sdk.V2Job{
			"my-job": {
				RunsOnRaw: json.RawMessage(`"my-model"`),
				Steps: []sdk.ActionStep{
					{ID: "my-step-with-hyphen", Run: "echo hello"},
				},
			},
		},
	}
	errs := validateWorkflowSchema(t, w)
	assert.Empty(t, errs, "step.id with hyphen should be accepted")
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

func TestWorkflowJsonSchema_StepIDWithUnderscore(t *testing.T) {
	w := sdk.V2Workflow{
		Name: "test-workflow",
		Jobs: map[string]sdk.V2Job{
			"myJob": {
				RunsOnRaw: json.RawMessage(`"my-model"`),
				Steps: []sdk.ActionStep{
					{ID: "my_step_with_underscore", Run: "echo hello"},
				},
			},
		},
	}
	errs := validateWorkflowSchema(t, w)
	assert.Empty(t, errs, "step.id with underscore should be accepted")
}

func TestWorkflowJsonSchema_StepIDWithSpace_ShouldFail(t *testing.T) {
	w := sdk.V2Workflow{
		Name: "test-workflow",
		Jobs: map[string]sdk.V2Job{
			"myJob": {
				RunsOnRaw: json.RawMessage(`"my-model"`),
				Steps: []sdk.ActionStep{
					{ID: "invalid step id", Run: "echo hello"},
				},
			},
		},
	}
	errs := validateWorkflowSchema(t, w)
	assert.NotEmpty(t, errs, "step.id with space should be rejected (must match ^[a-zA-Z0-9_-]{1,}$)")
}

func TestWorkflowJsonSchema_StepIDWithDot_ShouldFail(t *testing.T) {
	w := sdk.V2Workflow{
		Name: "test-workflow",
		Jobs: map[string]sdk.V2Job{
			"myJob": {
				RunsOnRaw: json.RawMessage(`"my-model"`),
				Steps: []sdk.ActionStep{
					{ID: "invalid.step.id", Run: "echo hello"},
				},
			},
		},
	}
	errs := validateWorkflowSchema(t, w)
	assert.NotEmpty(t, errs, "step.id with dot should be rejected (must match ^[a-zA-Z0-9_-]{1,}$)")
}

func TestWorkflowJsonSchema_InvalidJobKey_WithSpace(t *testing.T) {
	w := sdk.V2Workflow{
		Name: "test-workflow",
		Jobs: map[string]sdk.V2Job{
			"my job": {
				RunsOnRaw: json.RawMessage(`"my-model"`),
				Steps: []sdk.ActionStep{
					{Run: "echo hello"},
				},
			},
		},
	}
	errs := validateWorkflowSchema(t, w)
	assert.NotEmpty(t, errs, "job key with a space should be rejected by the schema (must match ^[a-zA-Z0-9_-]{1,}$)")
}

func TestWorkflowJsonSchema_InvalidJobKey_WithDot(t *testing.T) {
	w := sdk.V2Workflow{
		Name: "test-workflow",
		Jobs: map[string]sdk.V2Job{
			"my.job": {
				RunsOnRaw: json.RawMessage(`"my-model"`),
				Steps: []sdk.ActionStep{
					{Run: "echo hello"},
				},
			},
		},
	}
	errs := validateWorkflowSchema(t, w)
	assert.NotEmpty(t, errs, "job key with a dot should be rejected by the schema (must match ^[a-zA-Z0-9_-]{1,}$)")
}

func TestWorkflowJsonSchema_ValidJobKey(t *testing.T) {
	w := sdk.V2Workflow{
		Name: "test-workflow",
		Jobs: map[string]sdk.V2Job{
			"my_valid-Job1": {
				RunsOnRaw: json.RawMessage(`"my-model"`),
				Steps: []sdk.ActionStep{
					{Run: "echo hello"},
				},
			},
		},
	}
	errs := validateWorkflowSchema(t, w)
	assert.Empty(t, errs, "valid job key should be accepted")
}
