package api

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/sdk"
)

func Test_getWorkflowsHandler(t *testing.T) {
	// Init database
	api, db, router := newTestAPI(t)

	// Init user
	u, pass := assets.InsertAdminUser(api.mustDB(context.Background()))
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
	u, pass := assets.InsertAdminUser(api.mustDB(context.Background()))
	// Init project
	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, api.Cache, key, key, u)
	//Prepare request
	vars := map[string]string{
		"key":              proj.Key,
		"permWorkflowName": "workflow1",
	}
	uri := router.GetRoute("GET", api.getWorkflowHandler, vars)
	test.NotEmpty(t, uri)

	req := assets.NewAuthentifiedRequest(t, u, pass, "GET", uri, nil)
	//Do the request
	w := httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)
	assert.Equal(t, 404, w.Code)
}

func Test_getWorkflowHandler_withUsage(t *testing.T) {
	// Init database
	api, db, router := newTestAPI(t)

	// Init user
	u, pass := assets.InsertAdminUser(api.mustDB(context.Background()))
	// Init project
	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, api.Cache, key, key, u)
	//Prepare request
	vars := map[string]string{
		"key":              proj.Key,
		"permWorkflowName": "workflow1",
	}
	uri := router.GetRoute("GET", api.getWorkflowHandler, vars)
	test.NotEmpty(t, uri)

	pip := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip1",
		Type:       sdk.BuildPipeline,
	}

	test.NoError(t, pipeline.InsertPipeline(db, api.Cache, proj, &pip, u))

	proj, _ = project.LoadByID(db, api.Cache, proj.ID, u, project.LoadOptions.WithApplications, project.LoadOptions.WithPipelines, project.LoadOptions.WithEnvironments, project.LoadOptions.WithGroups)

	wf := sdk.Workflow{
		Name:       "workflow1",
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Root: &sdk.WorkflowNode{
			Pipeline: pip,
		},
	}

	test.NoError(t, workflow.Insert(db, api.Cache, &wf, proj, u))

	req := assets.NewAuthentifiedRequest(t, u, pass, "GET", uri+"?withUsage=true", nil)
	//Do the request
	w := httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)

	workflowResp := &sdk.Workflow{}
	test.NoError(t, json.Unmarshal(w.Body.Bytes(), workflowResp))

	assert.NotNil(t, workflowResp.Usage)
	assert.NotNil(t, workflowResp.Usage.Pipelines)
	assert.Equal(t, 1, len(workflowResp.Usage.Pipelines))
	assert.Equal(t, "pip1", workflowResp.Usage.Pipelines[0].Name)
}

func Test_postWorkflowHandlerWithoutRootShouldFail(t *testing.T) {
	// Init database
	api, db, router := newTestAPI(t)

	// Init user
	u, pass := assets.InsertAdminUser(api.mustDB(context.Background()))
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
	u, pass := assets.InsertAdminUser(api.mustDB(context.Background()))
	// Init project
	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, api.Cache, key, key, u)
	// Init pipeline
	pip := sdk.Pipeline{
		Name:      "pipeline1",
		ProjectID: proj.ID,
		Type:      sdk.BuildPipeline,
	}
	test.NoError(t, pipeline.InsertPipeline(api.mustDB(context.Background()), api.Cache, proj, &pip, nil))

	//Prepare request
	vars := map[string]string{
		"permProjectKey": proj.Key,
	}
	uri := router.GetRoute("POST", api.postWorkflowHandler, vars)
	test.NotEmpty(t, uri)

	// Insert application
	app := sdk.Application{
		Name:               "app1",
		RepositoryFullname: "test/app1",
		VCSServer:          "github",
	}
	test.NoError(t, application.Insert(api.mustDB(context.Background()), api.Cache, proj, &app, u))

	var workflow = &sdk.Workflow{
		Name:        "Name",
		Description: "Description",
		Root: &sdk.WorkflowNode{
			PipelineID: pip.ID,
			Context: &sdk.WorkflowNodeContext{
				ApplicationID: app.ID,
				Application:   &app,
			},
		},
	}

	req := assets.NewAuthentifiedRequest(t, u, pass, "POST", uri, &workflow)
	//Do the request
	w := httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)
	assert.Equal(t, 201, w.Code)

	test.NoError(t, json.Unmarshal(w.Body.Bytes(), &workflow))
	assert.NotEqual(t, 0, workflow.ID)

	assert.NotNil(t, workflow.Root.Context.Application)
	assert.NotNil(t, workflow.Root.Context.DefaultPayload)

	payload, err := workflow.Root.Context.DefaultPayloadToMap()
	test.NoError(t, err)

	assert.NotEmpty(t, payload["git.branch"], "git.branch should not be empty")
}

func Test_putWorkflowHandler(t *testing.T) {
	// Init database
	api, db, router := newTestAPI(t)

	// Init user
	u, pass := assets.InsertAdminUser(api.mustDB(context.Background()))
	// Init project
	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, api.Cache, key, key, u)

	// Init pipeline
	pip := sdk.Pipeline{
		Name:      "pipeline1",
		ProjectID: proj.ID,
		Type:      sdk.BuildPipeline,
	}
	test.NoError(t, pipeline.InsertPipeline(api.mustDB(context.Background()), api.Cache, proj, &pip, nil))

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
		"key":              proj.Key,
		"permWorkflowName": "Name",
	}
	uri = router.GetRoute("PUT", api.putWorkflowHandler, vars)
	test.NotEmpty(t, uri)

	// Insert application
	app := sdk.Application{
		Name:               "app1",
		RepositoryFullname: "test/app1",
		VCSServer:          "github",
	}
	test.NoError(t, application.Insert(api.mustDB(context.Background()), api.Cache, proj, &app, u))

	var workflow1 = &sdk.Workflow{
		Name:        "Name",
		Description: "Description 2",
		Root: &sdk.WorkflowNode{
			PipelineID: pip.ID,
			Context: &sdk.WorkflowNodeContext{
				ApplicationID: app.ID,
				Application:   &app,
			},
		},
	}

	req = assets.NewAuthentifiedRequest(t, u, pass, "PUT", uri, &workflow1)

	//Do the request
	w = httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)

	test.NoError(t, json.Unmarshal(w.Body.Bytes(), &workflow1))

	assert.NotEqual(t, 0, workflow1.ID)
	assert.Equal(t, "Description 2", workflow1.Description)

	assert.NotNil(t, workflow1.Root.Context.Application)
	assert.NotNil(t, workflow1.Root.Context.DefaultPayload)

	payload, err := workflow1.Root.Context.DefaultPayloadToMap()
	test.NoError(t, err)

	assert.NotEmpty(t, payload["git.branch"], "git.branch should not be empty")
}

func Test_deleteWorkflowHandler(t *testing.T) {
	// Init database
	api, db, router := newTestAPI(t)
	test.NoError(t, workflow.CreateBuiltinWorkflowHookModels(db))

	// Init user
	u, pass := assets.InsertAdminUser(api.mustDB(context.Background()))
	// Init project
	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, api.Cache, key, key, u)
	// Init pipeline
	pip := sdk.Pipeline{
		Name:      "pipeline1",
		ProjectID: proj.ID,
		Type:      sdk.BuildPipeline,
	}
	test.NoError(t, pipeline.InsertPipeline(api.mustDB(context.Background()), api.Cache, proj, &pip, nil))

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
		"key":              proj.Key,
		"permWorkflowName": "Name",
	}
	uri = router.GetRoute("DELETE", api.deleteWorkflowHandler, vars)
	test.NotEmpty(t, uri)

	req = assets.NewAuthentifiedRequest(t, u, pass, "DELETE", uri, nil)

	//Do the request
	w = httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)
}
