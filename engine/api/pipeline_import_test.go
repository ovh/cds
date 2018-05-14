package api

import (
	"encoding/json"
	"io/ioutil"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/sdk"
)

func Test_postPipelinePreviewHandler(t *testing.T) {
	api, db, _ := newTestAPI(t)
	u, pass := assets.InsertAdminUser(db)
	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10), u)
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
	req.Body = ioutil.NopCloser(strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-yaml")

	//Do the request
	rec := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	assert.Equal(t, 200, rec.Code)

	res, _ := ioutil.ReadAll(rec.Body)
	pip := &sdk.Pipeline{}
	test.NoError(t, json.Unmarshal(res, &pip))

	test.Equal(t, pip.Name, "echo")
	test.Equal(t, len(pip.Stages), 1)
	test.Equal(t, pip.Stages[0].Name, "Stage 1")
	test.Equal(t, len(pip.Stages[0].Jobs), 1)
	test.Equal(t, pip.Stages[0].Jobs[0].Action.Name, "echo with default")
}
