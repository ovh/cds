package workflowtemplate_test

import (
	"encoding/base64"
	"testing"

	"github.com/ovh/cds/engine/api/workflowtemplate"
	"github.com/ovh/cds/sdk"
	"github.com/stretchr/testify/assert"
)

func TestExecuteTemplate(t *testing.T) {
	tmpl := &sdk.WorkflowTemplate{
		ID: 42,
		Parameters: []sdk.WorkflowTemplateParameter{
			{Key: "name", Type: sdk.ParameterTypeString, Required: true},
			{Key: "withDeploy", Type: sdk.ParameterTypeBoolean, Required: true},
			{Key: "deployWhen", Type: sdk.ParameterTypeString},
		},
		Value: base64.StdEncoding.EncodeToString([]byte(`
name: [[.name]]
description: Test simple workflow
version: v1.0
workflow:
	Node-1:
		pipeline: Pipeline-1-[[.id]]
	[[if .params.withDeploy]]
	Node-2:
		depends_on:
		- Node-1
		when:
		- [[.params.deployWhen]]
		pipeline: Pipeline-1-[[.id]]
	[[end]]`)),
		Pipelines: []sdk.PipelineTemplate{{
			Value: base64.StdEncoding.EncodeToString([]byte(`
version: v1.0
name: Pipeline-1-[[.id]]
stages:
- Stage 1
jobs:
- job: Job 1
	stage: Stage 1
	steps:
	- script:
		- echo "Hello World!"
	- script:
		- echo "{{.cds.run.number}}"`)),
		}},
		Applications: []sdk.ApplicationTemplate{{
			Value: base64.StdEncoding.EncodeToString([]byte(`
version: v1.0
name: [[.name]]
vcs_server: [[.params.vcsServer]]
repo: [[.params.repo]]`)),
		}},
	}

	req := sdk.WorkflowTemplateRequest{
		Name: "my-workflow",
		Parameters: map[string]string{
			"withDeploy": "true",
			"deployWhen": "failure",
		},
	}

	res, err := workflowtemplate.Execute(tmpl, req)
	assert.Nil(t, err)

	t.Log(res.Workflow)
	for _, p := range res.Pipelines {
		t.Log(p)
	}
	for _, a := range res.Applications {
		t.Log(a)
	}

	assert.Equal(t, true, true)
}
