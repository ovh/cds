package api

import (
	"context"
	"encoding/json"
	"io"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/sdk"
)

func Test_postPipelinePreviewHandler(t *testing.T) {
	api, db, _ := newTestAPI(t)

	u, pass := assets.InsertAdminUser(t, db)
	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))
	test.NotNil(t, proj)

	//Prepare request
	vars := map[string]string{
		"permProjectKey": proj.Key,
	}
	uri := api.Router.GetRoute("POST", api.postPipelinePreviewHandler, vars)
	test.NotEmpty(t, uri)
	req := assets.NewAuthentifiedRequest(t, u, pass, "POST", uri+"?format=yaml", nil)

	body := `version: v1.0
name: echo
stages:
- Stage 1
jobs:
- job: echo with default
  stage: Stage 1
  steps:
  - script:
    - echo "test"
    - echo {{.limit | default ""}}`
	req.Body = io.NopCloser(strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-yaml")

	//Do the request
	rec := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	assert.Equal(t, 200, rec.Code)

	res, _ := io.ReadAll(rec.Body)
	pip := &sdk.Pipeline{}
	test.NoError(t, json.Unmarshal(res, &pip))

	test.Equal(t, pip.Name, "echo")
	test.Equal(t, len(pip.Stages), 1)
	test.Equal(t, pip.Stages[0].Name, "Stage 1")
	test.Equal(t, len(pip.Stages[0].Jobs), 1)
	test.Equal(t, pip.Stages[0].Jobs[0].Action.Name, "echo with default")
}

func Test_putPipelineImportHandler(t *testing.T) {
	api, db, _ := newTestAPI(t)

	u, pass := assets.InsertAdminUser(t, db)
	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))
	test.NotNil(t, proj)

	//Prepare request
	vars := map[string]string{
		"permProjectKey": proj.Key,
		"pipelineKey":    "testest",
	}
	uri := api.Router.GetRoute("PUT", api.putImportPipelineHandler, vars)
	test.NotEmpty(t, uri)
	req := assets.NewAuthentifiedRequest(t, u, pass, "PUT", uri+"?format=yaml", nil)

	body := `version: v1.0
name: echo
stages:
- Stage 1
jobs:
- job: echo with default
  stage: Stage 1
  steps:
  - script:
    - echo "test"
    - echo {{.limit | default ""}}`
	req.Body = io.NopCloser(strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-yaml")

	//Do the request
	rec := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	assert.Equal(t, 400, rec.Code)
}

func Test_putPipelineImportJSONWithoutVersionHandler(t *testing.T) {
	api, db, _ := newTestAPI(t)

	u, pass := assets.InsertAdminUser(t, db)
	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))
	test.NotNil(t, proj)

	//Prepare request
	vars := map[string]string{
		"permProjectKey": proj.Key,
		"pipelineKey":    "testest",
	}
	uri := api.Router.GetRoute("PUT", api.putImportPipelineHandler, vars)
	test.NotEmpty(t, uri)
	req := assets.NewAuthentifiedRequest(t, u, pass, "PUT", uri+"?format=json", nil)

	bodyjson := `{"name":"testest","stages":["Stage 1"],"jobs":[{"job":"echo with default","stage":"Stage 1","steps":[{"script":["echo \"test\"","echo {{.limit | default \"\"}}"]}]}]}`
	req.Body = io.NopCloser(strings.NewReader(bodyjson))
	req.Header.Set("Content-Type", "application/json")

	//Do the request
	rec := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	assert.Equal(t, 200, rec.Code)
}

func Test_putPipelineImportDifferentStageHandler(t *testing.T) {
	api, db, _ := newTestAPI(t)

	u, pass := assets.InsertAdminUser(t, db)
	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))
	test.NotNil(t, proj)

	//Prepare request
	vars := map[string]string{
		"permProjectKey": proj.Key,
		"pipelineKey":    "echo",
	}
	uri := api.Router.GetRoute("PUT", api.putImportPipelineHandler, vars)
	test.NotEmpty(t, uri)
	req := assets.NewAuthentifiedRequest(t, u, pass, "PUT", uri+"?format=yaml", nil)

	body := `version: v1.0
name: echo
stages:
- Stage 1
options:
  Stage 1:
    conditions:
      check:
      - variable: git.branch
        operator: ne
        value: ""
jobs:
- job: New Job`
	req.Body = io.NopCloser(strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-yaml")

	//Do the request
	rec := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	assert.Equal(t, 200, rec.Code)

	uri = api.Router.GetRoute("PUT", api.putImportPipelineHandler, vars)
	test.NotEmpty(t, uri)
	req = assets.NewAuthentifiedRequest(t, u, pass, "PUT", uri+"?format=yaml", nil)

	body = `version: v1.0
name: echo
stages:
- Stage 0
- Stage 1
jobs:
- job: JobPushBuild
  stage: Stage 1
  steps:
  - pushBuildInfo: '{{.cds.workflow}}'
  - script:
    - echo "coucou"
- job: Echo
  stage: Stage 0
  steps:
  - script:
    - echo "coucou"
`
	req.Body = io.NopCloser(strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-yaml")

	//Do the request
	rec = httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	assert.Equal(t, 200, rec.Code)

	pip, err := pipeline.LoadPipeline(context.TODO(), db, proj.Key, "echo", true)
	test.NoError(t, err)

	assert.Len(t, pip.Stages, 2)
	assert.Equal(t, 1, pip.Stages[0].BuildOrder)
	assert.Equal(t, "Stage 0", pip.Stages[0].Name)
	assert.Equal(t, 2, pip.Stages[1].BuildOrder)
	assert.Equal(t, "Stage 1", pip.Stages[1].Name)

	assert.Equal(t, "PushBuildInfo", pip.Stages[1].Jobs[0].Action.Actions[0].Name)
}
