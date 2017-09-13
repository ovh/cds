package api

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/sdk"
)

func Test_getWorkflowsHandler(t *testing.T) {
	// Init database
	api, db, router := newTestAPI(t)

	

	
	// Init user
	u, pass := assets.InsertAdminUser(api.mustDB())
	// Init project
	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, api.Cache, key, key, u)
	//Prepare request
	vars := map[string]string{
		"permProjectKey": proj.Key,
	}
	uri := router.GetRoute("GET", api.getWorkflowsHandler, vars)
	test.NotEmpty(t, uri)
	req := assets.NewAuthentifiedRequest(t, u, pass, "GET", uri, vars)

	//Do the request
	w := httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)
}

func Test_getWorkflowHandler(t *testing.T) {
	// Init database
	api, db, router := newTestAPI(t)

	

	
	// Init user
	u, pass := assets.InsertAdminUser(api.mustDB())
	// Init project
	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, api.Cache, key, key, u)
	//Prepare request
	vars := map[string]string{
		"permProjectKey": proj.Key,
		"workflowName":   "workflow1",
	}
	uri := router.GetRoute("GET", api.getWorkflowHandler, vars)
	test.NotEmpty(t, uri)

	req := assets.NewAuthentifiedRequest(t, u, pass, "GET", uri, nil)
	//Do the request
	w := httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)
	assert.Equal(t, 404, w.Code)
}

func Test_postWorkflowHandlerWithoutRootShouldFail(t *testing.T) {
	// Init database
	api, db, router := newTestAPI(t)
	

	
	// Init user
	u, pass := assets.InsertAdminUser(api.mustDB())
	// Init project
	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, api.Cache, key, key, u)
	//Prepare request
	vars := map[string]string{
		"permProjectKey": proj.Key,
	}
	uri := router.GetRoute("POST", api.postWorkflowHandler, vars)
	test.NotEmpty(t, uri)

	var workflowResponse sdk.Workflow
	req := assets.NewAuthentifiedRequest(t, u, pass, "POST", uri, &workflowResponse)
	//Do the request
	w := httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)
	assert.Equal(t, 400, w.Code)
}

func Test_postWorkflowHandlerWithRootShouldSuccess(t *testing.T) {
	// Init database
	api, db, router := newTestAPI(t)
	

	
	// Init user
	u, pass := assets.InsertAdminUser(api.mustDB())
	// Init project
	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, api.Cache, key, key, u)
	// Init pipeline
	pip := sdk.Pipeline{
		Name:      "pipeline1",
		ProjectID: proj.ID,
		Type:      sdk.BuildPipeline,
	}
	test.NoError(t, pipeline.InsertPipeline(api.mustDB(), proj, &pip, nil))

	//Prepare request
	vars := map[string]string{
		"permProjectKey": proj.Key,
	}
	uri := router.GetRoute("POST", api.postWorkflowHandler, vars)
	test.NotEmpty(t, uri)

	var workflow = &sdk.Workflow{
		Name:        "Name",
		Description: "Description",
		Root: &sdk.WorkflowNode{
			PipelineID: pip.ID,
		},
	}

	req := assets.NewAuthentifiedRequest(t, u, pass, "POST", uri, &workflow)
	//Do the request
	w := httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)
	assert.Equal(t, 201, w.Code)

	test.NoError(t, json.Unmarshal(w.Body.Bytes(), &workflow))
	assert.NotEqual(t, 0, workflow.ID)
}

func Test_putWorkflowHandler(t *testing.T) {
	// Init database
	api, db, router := newTestAPI(t)
	

	
	// Init user
	u, pass := assets.InsertAdminUser(api.mustDB())
	// Init project
	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, api.Cache, key, key, u)
	// Init pipeline
	pip := sdk.Pipeline{
		Name:      "pipeline1",
		ProjectID: proj.ID,
		Type:      sdk.BuildPipeline,
	}
	test.NoError(t, pipeline.InsertPipeline(api.mustDB(), proj, &pip, nil))

	//Prepare request
	vars := map[string]string{
		"permProjectKey": proj.Key,
	}
	uri := router.GetRoute("POST", api.postWorkflowHandler, vars)
	test.NotEmpty(t, uri)

	var workflow = &sdk.Workflow{
		Name:        "Name",
		Description: "Description",
		Root: &sdk.WorkflowNode{
			PipelineID: pip.ID,
		},
	}

	req := assets.NewAuthentifiedRequest(t, u, pass, "POST", uri, &workflow)

	//Do the request
	w := httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)
	assert.Equal(t, 201, w.Code)
	test.NoError(t, json.Unmarshal(w.Body.Bytes(), &workflow))

	//Prepare request
	vars = map[string]string{
		"permProjectKey": proj.Key,
		"workflowName":   "Name",
	}
	uri = router.GetRoute("PUT", api.putWorkflowHandler, vars)
	test.NotEmpty(t, uri)

	var workflow1 = &sdk.Workflow{
		Name:        "Name 2",
		Description: "Description 2",
		Root: &sdk.WorkflowNode{
			PipelineID: pip.ID,
		},
	}

	req = assets.NewAuthentifiedRequest(t, u, pass, "PUT", uri, &workflow1)

	//Do the request
	w = httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)

	test.NoError(t, json.Unmarshal(w.Body.Bytes(), &workflow1))

	assert.NotEqual(t, 0, workflow1.ID)
	assert.Equal(t, "Name 2", workflow1.Name)
}

func Test_deleteWorkflowHandler(t *testing.T) {
	// Init database
	api, db, router := newTestAPI(t)
	

	
	// Init user
	u, pass := assets.InsertAdminUser(api.mustDB())
	// Init project
	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, api.Cache, key, key, u)
	// Init pipeline
	pip := sdk.Pipeline{
		Name:      "pipeline1",
		ProjectID: proj.ID,
		Type:      sdk.BuildPipeline,
	}
	test.NoError(t, pipeline.InsertPipeline(api.mustDB(), proj, &pip, nil))

	//Prepare request
	vars := map[string]string{
		"permProjectKey": proj.Key,
	}
	uri := router.GetRoute("POST", api.postWorkflowHandler, vars)
	test.NotEmpty(t, uri)

	var workflow = &sdk.Workflow{
		Name:        "Name",
		Description: "Description",
		Root: &sdk.WorkflowNode{
			PipelineID: pip.ID,
		},
	}

	req := assets.NewAuthentifiedRequest(t, u, pass, "POST", uri, &workflow)

	//Do the request
	w := httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)
	assert.Equal(t, 201, w.Code)
	test.NoError(t, json.Unmarshal(w.Body.Bytes(), &workflow))

	vars = map[string]string{
		"permProjectKey": proj.Key,
		"workflowName":   "Name",
	}
	uri = router.GetRoute("DELETE", api.deleteWorkflowHandler, vars)
	test.NotEmpty(t, uri)

	req = assets.NewAuthentifiedRequest(t, u, pass, "DELETE", uri, nil)

	//Do the request
	w = httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)
}
