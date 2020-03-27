package workflowtemplate_test

import (
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"

	"github.com/ovh/cds/engine/api/workflowtemplate"
	"github.com/ovh/cds/sdk"
)

func TestExecuteTemplate(t *testing.T) {
	tmpl := sdk.WorkflowTemplate{
		ID: 42,
		Parameters: []sdk.WorkflowTemplateParameter{
			{Key: "withDeploy", Type: sdk.ParameterTypeBoolean, Required: true},
			{Key: "deployWhen", Type: sdk.ParameterTypeString},
			{Key: "repo", Type: sdk.ParameterTypeRepository},
			{Key: "object", Type: sdk.ParameterTypeJSON},
			{Key: "list", Type: sdk.ParameterTypeJSON},
		},
		Workflow: base64.StdEncoding.EncodeToString([]byte(`
name: [[.name]]
description: Test simple workflow
version: v1.0
workflow:
  Node-1:
    pipeline: Pipeline-[[.id]]
  [[if .params.withDeploy -]]
  Node-2:
    depends_on:
    - Node-1
    when:
    - [[.params.deployWhen]]
    pipeline: Pipeline-[[.id]]
  [[- end -]]`)),
		Pipelines: []sdk.PipelineTemplate{{
			Value: base64.StdEncoding.EncodeToString([]byte(`
version: v1.0
name: Pipeline-[[.id]]
stages:
- Stage 1
jobs:
- job: Job 1
  stage: Stage 1
  steps:
  - script:
    - echo "Hello World!"
    - echo "[[ "Hello World Lower!" | lower ]]"
    - echo "[[.params.object.key1]]"
    - echo "[[range .params.list]][[.]][[end]]"
  - script:
    - echo "{{.cds.run.number}}"`)),
		}},
		Applications: []sdk.ApplicationTemplate{{
			Value: base64.StdEncoding.EncodeToString([]byte(`
version: v1.0
name: [[.name]]
vcs_server: [[.params.repo.vcs]]
repo: [[.params.repo.repository]]`)),
		}},
		Environments: []sdk.EnvironmentTemplate{{
			Value: base64.StdEncoding.EncodeToString([]byte(`
name: Environment-[[.id]]
values:
  key1:
    type: string
    value: value1`)),
		}},
	}

	instance := sdk.WorkflowTemplateInstance{
		ID: 5,
		Request: sdk.WorkflowTemplateRequest{
			WorkflowName: "my-workflow",
			Parameters: map[string]string{
				"withDeploy": "true",
				"deployWhen": "failure",
				"repo":       "github/ovh/cds",
				"object":     "{\"key1\":\"value1\"}",
				"list":       "[\"value1\",\"value2\"]",
			},
		},
	}

	res, err := workflowtemplate.Execute(tmpl, instance)
	assert.Nil(t, err)

	buf, err := yaml.Marshal(res.Workflow)
	require.NoError(t, err)
	assert.Equal(t, `name: my-workflow
description: Test simple workflow
version: v1.0
workflow:
  Node-1:
    pipeline: Pipeline-5
  Node-2:
    depends_on:
    - Node-1
    when:
    - failure
    pipeline: Pipeline-5
`, string(buf))

	assert.Equal(t, 1, len(res.Pipelines))
	buf, err = yaml.Marshal(res.Pipelines[0])
	require.NoError(t, err)
	assert.Equal(t, `version: v1.0
name: Pipeline-5
stages:
- Stage 1
jobs:
- job: Job 1
  stage: Stage 1
  steps:
  - script:
    - echo "Hello World!"
    - echo "hello world lower!"
    - echo "value1"
    - echo "value1value2"
  - script:
    - echo "{{.cds.run.number}}"
`, string(buf))

	assert.Equal(t, 1, len(res.Applications))
	buf, err = yaml.Marshal(res.Applications[0])
	require.NoError(t, err)
	assert.Equal(t, `version: v1.0
name: my-workflow
vcs_server: github
repo: ovh/cds
`, string(buf))

	assert.Equal(t, 1, len(res.Environments))
	buf, err = yaml.Marshal(res.Environments[0])
	require.NoError(t, err)
	assert.Equal(t, `name: Environment-5
values:
  key1:
    type: string
    value: value1
`, string(buf))
}

func TestExecuteTemplateWithError(t *testing.T) {
	tmpl := sdk.WorkflowTemplate{
		ID: 42,
		Parameters: []sdk.WorkflowTemplateParameter{
			{Key: "withDeploy", Type: sdk.ParameterTypeBoolean, Required: true},
			{Key: "deployWhen", Type: sdk.ParameterTypeString},
			{Key: "repo", Type: sdk.ParameterTypeRepository},
		},
		Workflow: base64.StdEncoding.EncodeToString([]byte(`
name: [[.name]
description: Test simple workflow with error
version: v1.0`)),
		Pipelines: []sdk.PipelineTemplate{{
			Value: base64.StdEncoding.EncodeToString([]byte(`
version: v1.0
name: Pipeline-[[error .id]]
stages:
- Stage 1`)),
		}},
		Applications: []sdk.ApplicationTemplate{{
			Value: base64.StdEncoding.EncodeToString([]byte(`
version: v1.0
name: [[`)),
		}},
		Environments: []sdk.EnvironmentTemplate{{
			Value: base64.StdEncoding.EncodeToString([]byte(`
name: Environment-[[if .id]]`)),
		}},
	}

	_, err := workflowtemplate.Parse(tmpl)
	assert.NotNil(t, err)
	e := sdk.ExtractHTTPError(err, "")
	assert.Equal(t, sdk.ErrCannotParseTemplate.ID, e.ID)
	errs := []sdk.WorkflowTemplateError{{
		Type:    "workflow",
		Line:    2,
		Message: "unexpected \"]\" in operand",
	}, {
		Type:    "pipeline",
		Line:    3,
		Message: "function \"error\" not defined",
	}, {
		Type:    "application",
		Line:    3,
		Message: "unexpected unclosed action in command",
	}, {
		Type:    "environment",
		Line:    2,
		Message: "unexpected EOF",
	}}
	assert.Equal(t, errs, e.Data)
}
