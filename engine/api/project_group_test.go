package api

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ovh/cds/engine/api/bootstrap"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/sdk"
)

// Test_ProjectPerms Useful to test permission on project
func Test_ProjectPerms(t *testing.T) {
	api, db, router, end := newTestAPI(t, bootstrap.InitiliazeDB)
	defer end()
	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10), nil)
	u, pass := assets.InsertLambdaUser(api.mustDB(), &proj.ProjectGroups[0].Group)

	//First pipeline
	pip := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip1",
		Type:       sdk.BuildPipeline,
	}
	test.NoError(t, pipeline.InsertPipeline(api.mustDB(), api.Cache, proj, &pip, u))

	newWf := sdk.Workflow{
		Name: sdk.RandomString(10),
		WorkflowData: &sdk.WorkflowData{
			Node: sdk.Node{
				Name: "root",
				Type: sdk.NodeTypePipeline,
				Context: &sdk.NodeContext{
					PipelineID: pip.ID,
				},
			},
		},

		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
	}

	(&newWf).RetroMigrate()

	//Prepare request
	vars := map[string]string{
		"permProjectKey": proj.Key,
	}
	uri := router.GetRoute("POST", api.postWorkflowHandler, vars)
	test.NotEmpty(t, uri)
	req := assets.NewAuthentifiedRequest(t, u, pass, "POST", uri, &newWf)
	//Do the request
	w := httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)
	assert.Equal(t, 201, w.Code)

	test.NoError(t, json.Unmarshal(w.Body.Bytes(), &newWf))
	assert.NotEqual(t, 0, newWf.ID)
	newEnv := sdk.Environment{
		Name: "env-" + sdk.RandomString(5),
	}
	uri = router.GetRoute("POST", api.addEnvironmentHandler, vars)
	test.NotEmpty(t, uri)
	req = assets.NewAuthentifiedRequest(t, u, pass, "POST", uri, &newEnv)
	//Do the request
	w = httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)

	newApp := sdk.Application{
		Name: "app-" + sdk.RandomString(5),
	}
	uri = router.GetRoute("POST", api.addApplicationHandler, vars)
	test.NotEmpty(t, uri)
	req = assets.NewAuthentifiedRequest(t, u, pass, "POST", uri, &newApp)
	//Do the request
	w = httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)

	newPip := sdk.Pipeline{
		Name: "pip-" + sdk.RandomString(5),
		Type: sdk.BuildPipeline,
	}
	uri = router.GetRoute("POST", api.addPipelineHandler, vars)
	test.NotEmpty(t, uri)
	req = assets.NewAuthentifiedRequest(t, u, pass, "POST", uri, &newPip)
	//Do the request
	w = httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)
}
