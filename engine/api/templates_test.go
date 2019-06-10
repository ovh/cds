package api

import (
	"encoding/base64"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ovh/cds/engine/api/bootstrap"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/engine/api/workflowtemplate"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/slug"
)

func Test_postTemplateApplyHandler(t *testing.T) {
	api, db, _, end := newTestAPI(t, bootstrap.InitiliazeDB)
	defer end()

	u, pass := assets.InsertAdminUser(api.mustDB())
	g, err := group.LoadGroup(api.mustDB(), "shared.infra")
	assert.NoError(t, err)

	name := sdk.RandomString(10)
	pipelineName := sdk.RandomString(10)
	template := &sdk.WorkflowTemplate{
		GroupID: g.ID,
		Name:    name,
		Slug:    slug.Convert(name),
		Workflow: base64.StdEncoding.EncodeToString([]byte(
			`name: [[.name]]
version: v1.0
workflow:
  Node-1:
    pipeline: ` + pipelineName,
		)),
		Pipelines: []sdk.PipelineTemplate{{
			Value: base64.StdEncoding.EncodeToString([]byte(
				`version: v1.0
name: ` + pipelineName + `
stages:
- Stage 1
jobs:
- job: Job 1
  stage: Stage 1
  steps:
  - script:
    - echo "Hello World!"`,
			)),
		}},
	}
	assert.NoError(t, workflowtemplate.Insert(db, template))

	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10), u)

	// prepare the request
	uri := api.Router.GetRoute("POST", api.postTemplateApplyHandler, map[string]string{
		"groupName":        g.Name,
		"permTemplateSlug": template.Slug,
	})
	test.NotEmpty(t, uri)

	wtr := sdk.WorkflowTemplateRequest{
		ProjectKey:   proj.Key,
		WorkflowName: sdk.RandomString(10),
	}
	req := assets.NewAuthentifiedRequest(t, u, pass, "POST", uri+"?import=true", wtr)

	// execute the request
	rec := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)

	// check result
	assert.Equal(t, 200, rec.Code)
	assert.Equal(t, wtr.WorkflowName, rec.Header().Get(sdk.ResponseWorkflowNameHeader))

	v, err := json.Marshal([]string{"Pipeline " + pipelineName + " successfully created", "Workflow " + wtr.WorkflowName + " has been created"})
	assert.NoError(t, err)

	assert.Equal(t, string(v), rec.Body.String())
}

func Test_postTemplateBulkHandler(t *testing.T) {
	api, db, _, end := newTestAPI(t, bootstrap.InitiliazeDB)
	defer end()

	u, pass := assets.InsertAdminUser(api.mustDB())
	g, err := group.LoadGroup(api.mustDB(), "shared.infra")
	assert.NoError(t, err)

	name := sdk.RandomString(10)
	pipelineName := sdk.RandomString(10)
	template := &sdk.WorkflowTemplate{
		GroupID: g.ID,
		Name:    name,
		Slug:    slug.Convert(name),
		Workflow: base64.StdEncoding.EncodeToString([]byte(
			`name: [[.name]]
version: v1.0
workflow:
  Node-1:
    pipeline: ` + pipelineName,
		)),
		Pipelines: []sdk.PipelineTemplate{{
			Value: base64.StdEncoding.EncodeToString([]byte(
				`version: v1.0
name: ` + pipelineName + `
stages:
- Stage 1
jobs:
- job: Job 1
  stage: Stage 1
  steps:
  - script:
    - echo "Hello World!"`,
			)),
		}},
	}
	assert.NoError(t, workflowtemplate.Insert(db, template))

	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10), u)

	// prepare the request
	uri := api.Router.GetRoute("POST", api.postTemplateBulkHandler, map[string]string{
		"groupName":        g.Name,
		"permTemplateSlug": template.Slug,
	})
	test.NotEmpty(t, uri)

	wtb := sdk.WorkflowTemplateBulk{
		Operations: []sdk.WorkflowTemplateBulkOperation{{
			Request: sdk.WorkflowTemplateRequest{
				ProjectKey:   proj.Key,
				WorkflowName: sdk.RandomString(10),
			},
		}, {
			Request: sdk.WorkflowTemplateRequest{
				ProjectKey:   proj.Key,
				WorkflowName: sdk.RandomString(10),
			},
		}},
	}
	req := assets.NewAuthentifiedRequest(t, u, pass, "POST", uri, wtb)

	// execute the request
	rec := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)

	// check result
	assert.Equal(t, 200, rec.Code)

	var result sdk.WorkflowTemplateBulk
	assert.NoError(t, json.Unmarshal(rec.Body.Bytes(), &result))

	assert.Equal(t, 2, len(result.Operations))
}
