package api

import (
	"io/ioutil"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/fsamin/go-dump"
	"github.com/stretchr/testify/assert"

	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/sdk"
)

func Test_postWorkflowImportHandler(t *testing.T) {
	api, db, _ := newTestAPI(t)
	u, pass := assets.InsertAdminUser(db)
	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10), u)
	test.NotNil(t, proj)
	pip := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip1",
		Type:       sdk.BuildPipeline,
	}
	test.NoError(t, pipeline.InsertPipeline(db, api.Cache, proj, &pip, u))

	//Prepare request
	vars := map[string]string{
		"permProjectKey": proj.Key,
	}
	uri := api.Router.GetRoute("POST", api.postWorkflowImportHandler, vars)
	test.NotEmpty(t, uri)
	req := assets.NewAuthentifiedRequest(t, u, pass, "POST", uri, nil)

	body := `name: test_1
version: v1.0
workflow:
  pip1:
    pipeline: pip1
  pip1_2:
    depends_on:
      - pip1
    pipeline: pip1`
	req.Body = ioutil.NopCloser(strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-yaml")

	//Do the request
	rec := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	assert.Equal(t, 200, rec.Code)

	//Check result
	t.Logf(">>%s", rec.Body.String())

	w, err := workflow.Load(db, api.Cache, proj.Key, "test_1", u)
	test.NoError(t, err)

	assert.NotNil(t, w)

	m, _ := dump.ToStringMap(w)
	dump.Dump(m)
	assert.Equal(t, "test_1", m["Workflow.Name"])
	assert.Equal(t, "pip1", m["Workflow.Root.Name"])
	assert.Equal(t, "pip1", m["Workflow.Root.Pipeline.Name"])
	assert.Equal(t, "pip1_2", m["Workflow.Root.Triggers.Triggers0.WorkflowDestNode.Name"])
	assert.Equal(t, "pip1", m["Workflow.Root.Triggers.Triggers0.WorkflowDestNode.Pipeline.Name"])

}
