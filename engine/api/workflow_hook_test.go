package api

import (
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/sdk"
)

func Test_getWorkflowHookModelsHandlerAsLambdaUser(t *testing.T) {
	api, _, _ := newTestAPI(t)
	db := api.mustDB()
	cache := api.Cache
	test.NoError(t, workflow.CreateBuiltinWorkflowHookModels(api.mustDB()))
	u, passUser := assets.InsertLambdaUser(api.mustDB())

	proj := assets.InsertTestProject(t, db, cache, sdk.RandomString(10), sdk.RandomString(10), u)
	test.NoError(t, group.InsertUserInGroup(db, proj.ProjectGroups[0].Group.ID, u.ID, true))

	pip := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       sdk.RandomString(10),
		Type:       sdk.BuildPipeline,
	}

	test.NoError(t, pipeline.InsertPipeline(db, cache, proj, &pip, u))
	test.NoError(t, group.InsertGroupInPipeline(db, pip.ID, proj.ProjectGroups[0].Group.ID, 7))

	loadUserPermissions(db, cache, u)
	proj, _ = project.LoadByID(db, cache, proj.ID, u, project.LoadOptions.WithApplications,
		project.LoadOptions.WithPipelines, project.LoadOptions.WithEnvironments, project.LoadOptions.WithGroups)

	w := sdk.Workflow{
		Name:       sdk.RandomString(10),
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Root: &sdk.WorkflowNode{
			Pipeline: pip,
		},
	}

	test.NoError(t, workflow.Insert(db, cache, &w, proj, u))

	//Prepare request
	vars := map[string]string{}
	vars["key"] = proj.Key
	vars["permWorkflowName"] = w.Name
	vars["nodeID"] = fmt.Sprintf("%d", w.Root.ID)

	uri := api.Router.GetRoute("GET", api.getWorkflowHookModelsHandler, vars)
	test.NotEmpty(t, uri)
	req := assets.NewAuthentifiedRequest(t, u, passUser, "GET", uri, nil)

	//Do the request
	rec := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	assert.Equal(t, 200, rec.Code)

	//Check result
	models := []sdk.WorkflowHookModel{}
	test.NoError(t, json.Unmarshal(rec.Body.Bytes(), &models))
	assert.Len(t, models, 2, "")
}

func Test_getWorkflowHookModelsHandlerAsAdminUser(t *testing.T) {
	api, _, _ := newTestAPI(t)
	db := api.mustDB()
	cache := api.Cache
	test.NoError(t, workflow.CreateBuiltinWorkflowHookModels(api.mustDB()))
	admin, passAdmin := assets.InsertAdminUser(api.mustDB())

	proj := assets.InsertTestProject(t, db, cache, sdk.RandomString(10), sdk.RandomString(10), admin)

	pip := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       sdk.RandomString(10),
		Type:       sdk.BuildPipeline,
	}

	test.NoError(t, pipeline.InsertPipeline(db, cache, proj, &pip, admin))

	app := sdk.Application{
		Name:               sdk.RandomString(10),
		ProjectKey:         proj.Key,
		ProjectID:          proj.ID,
		RepositoryFullname: "ovh/cds",
	}
	test.NoError(t, application.Insert(db, cache, proj, &app, admin))

	proj, _ = project.LoadByID(db, cache, proj.ID, admin, project.LoadOptions.WithApplications,
		project.LoadOptions.WithPipelines, project.LoadOptions.WithEnvironments, project.LoadOptions.WithGroups)

	w := sdk.Workflow{
		Name:       sdk.RandomString(10),
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Root: &sdk.WorkflowNode{
			Pipeline: pip,
			Context: &sdk.WorkflowNodeContext{
				Application: &app,
			},
		},
	}

	test.NoError(t, workflow.Insert(db, cache, &w, proj, admin))

	//Prepare request
	vars := map[string]string{}
	vars["key"] = proj.Key
	vars["permWorkflowName"] = w.Name
	vars["nodeID"] = fmt.Sprintf("%d", w.Root.ID)

	uri := api.Router.GetRoute("GET", api.getWorkflowHookModelsHandler, vars)
	test.NotEmpty(t, uri)
	req := assets.NewAuthentifiedRequest(t, admin, passAdmin, "GET", uri, nil)

	//Do the request
	rec := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	assert.Equal(t, 200, rec.Code)

	//Check result
	models := []sdk.WorkflowHookModel{}
	test.NoError(t, json.Unmarshal(rec.Body.Bytes(), &models))
	assert.Len(t, models, 2, "")
}

func Test_getWorkflowHookModelHandler(t *testing.T) {
	api, _, _ := newTestAPI(t)
	test.NoError(t, workflow.CreateBuiltinWorkflowHookModels(api.mustDB()))
	admin, passAdmin := assets.InsertAdminUser(api.mustDB())

	//Prepare request
	vars := map[string]string{
		"model": sdk.WebHookModelName,
	}
	uri := api.Router.GetRoute("GET", api.getWorkflowHookModelHandler, vars)
	test.NotEmpty(t, uri)
	req := assets.NewAuthentifiedRequest(t, admin, passAdmin, "GET", uri, nil)

	//Do the request
	rec := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)

	//Check result
	model := sdk.WorkflowHookModel{}
	test.NoError(t, json.Unmarshal(rec.Body.Bytes(), &model))
	assert.Equal(t, sdk.WebHookModelName, model.Name)
	assert.Equal(t, sdk.WebHookModel.Command, model.Command)
	assert.Equal(t, sdk.WebHookModel.Description, model.Description)
	assert.Equal(t, sdk.WebHookModel.Disabled, model.Disabled)
	assert.Len(t, model.DefaultConfig, len(sdk.WebHookModel.DefaultConfig))
}

func Test_putWorkflowHookModelHandlerAsAdminUser(t *testing.T) {
	api, _, _ := newTestAPI(t)
	test.NoError(t, workflow.CreateBuiltinWorkflowHookModels(api.mustDB()))
	admin, passAdmin := assets.InsertAdminUser(api.mustDB())

	//Prepare request
	vars := map[string]string{
		"model": sdk.WebHookModelName,
	}
	uri := api.Router.GetRoute("GET", api.getWorkflowHookModelHandler, vars)
	test.NotEmpty(t, uri)
	req := assets.NewAuthentifiedRequest(t, admin, passAdmin, "GET", uri, nil)

	//Do the request
	rec := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)

	//Check result
	model := sdk.WorkflowHookModel{}
	test.NoError(t, json.Unmarshal(rec.Body.Bytes(), &model))
	assert.Equal(t, sdk.WebHookModelName, model.Name)

	//Now update it
	model.Disabled = false

	//Prepare the request
	uri = api.Router.GetRoute("PUT", api.putWorkflowHookModelHandler, vars)
	test.NotEmpty(t, uri)
	req = assets.NewAuthentifiedRequest(t, admin, passAdmin, "PUT", uri, model)

	//Do the request
	rec = httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)

	//Check result
	model = sdk.WorkflowHookModel{}
	test.NoError(t, json.Unmarshal(rec.Body.Bytes(), &model))
	assert.Equal(t, sdk.WebHookModelName, model.Name)
	assert.Equal(t, false, model.Disabled)

}

func Test_putWorkflowHookModelHandlerAsLambdaUser(t *testing.T) {
	api, _, _ := newTestAPI(t)
	test.NoError(t, workflow.CreateBuiltinWorkflowHookModels(api.mustDB()))

	u, pass := assets.InsertLambdaUser(api.mustDB())

	//Prepare request
	vars := map[string]string{
		"model": sdk.WebHookModelName,
	}
	uri := api.Router.GetRoute("GET", api.getWorkflowHookModelHandler, vars)
	test.NotEmpty(t, uri)
	req := assets.NewAuthentifiedRequest(t, u, pass, "GET", uri, nil)

	//Do the request
	rec := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)

	//Check result
	model := sdk.WorkflowHookModel{}
	test.NoError(t, json.Unmarshal(rec.Body.Bytes(), &model))
	assert.Equal(t, sdk.WebHookModelName, model.Name)

	//Now update it
	model.Disabled = false

	//Prepare the request
	uri = api.Router.GetRoute("PUT", api.putWorkflowHookModelHandler, vars)
	test.NotEmpty(t, uri)
	req = assets.NewAuthentifiedRequest(t, u, pass, "PUT", uri, model)

	//Do the request
	rec = httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)

	assert.Equal(t, 403, rec.Code)

}
