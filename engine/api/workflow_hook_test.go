package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

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
	api, _, _, end := newTestAPI(t)
	defer end()
	db := api.mustDB()
	cache := api.Cache
	test.NoError(t, workflow.CreateBuiltinWorkflowHookModels(api.mustDB()))
	u, passUser := assets.InsertLambdaUser(t, api.mustDB())

	proj := assets.InsertTestProject(t, db, cache, sdk.RandomString(10), sdk.RandomString(10))
	require.NoError(t, group.InsertLinkGroupUser(context.TODO(), db, &group.LinkGroupUser{
		GroupID:            proj.ProjectGroups[0].Group.ID,
		AuthentifiedUserID: u.ID,
		Admin:              true,
	}))

	pip := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       sdk.RandomString(10),
	}

	test.NoError(t, pipeline.InsertPipeline(db, cache, proj, &pip))

	proj, _ = project.LoadByID(db, cache, proj.ID,
		project.LoadOptions.WithApplications,
		project.LoadOptions.WithPipelines,
		project.LoadOptions.WithEnvironments,
		project.LoadOptions.WithGroups,
	)

	w := sdk.Workflow{
		Name:       sdk.RandomString(10),
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		WorkflowData: &sdk.WorkflowData{
			Node: sdk.Node{
				Name: "root",
				Type: sdk.NodeTypePipeline,
				Context: &sdk.NodeContext{
					PipelineID: pip.ID,
				},
			},
		},
	}

	test.NoError(t, workflow.Insert(context.TODO(), db, cache, &w, proj))

	//Prepare request
	vars := map[string]string{}
	vars["key"] = proj.Key
	vars["permWorkflowName"] = w.Name
	vars["nodeID"] = fmt.Sprintf("%d", w.WorkflowData.Node.ID)

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
	assert.Len(t, models, 4, "")
}

func Test_getWorkflowHookModelsHandlerAsAdminUser(t *testing.T) {
	api, _, _, end := newTestAPI(t)
	defer end()
	db := api.mustDB()
	cache := api.Cache
	test.NoError(t, workflow.CreateBuiltinWorkflowHookModels(api.mustDB()))
	admin, passAdmin := assets.InsertAdminUser(t, api.mustDB())

	proj := assets.InsertTestProject(t, db, cache, sdk.RandomString(10), sdk.RandomString(10))

	pip := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       sdk.RandomString(10),
	}

	test.NoError(t, pipeline.InsertPipeline(db, cache, proj, &pip))

	app := sdk.Application{
		Name:               sdk.RandomString(10),
		ProjectKey:         proj.Key,
		ProjectID:          proj.ID,
		RepositoryFullname: "ovh/cds",
	}
	test.NoError(t, application.Insert(db, cache, proj, &app))

	proj, _ = project.LoadByID(db, cache, proj.ID, project.LoadOptions.WithApplications,
		project.LoadOptions.WithPipelines, project.LoadOptions.WithEnvironments, project.LoadOptions.WithGroups)

	w := sdk.Workflow{
		Name:       sdk.RandomString(10),
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		WorkflowData: &sdk.WorkflowData{
			Node: sdk.Node{
				Name: "root",
				Type: sdk.NodeTypePipeline,
				Context: &sdk.NodeContext{
					ApplicationID: app.ID,
					PipelineID:    pip.ID,
				},
			},
		},
	}

	test.NoError(t, workflow.Insert(context.TODO(), db, cache, &w, proj))

	//Prepare request
	vars := map[string]string{}
	vars["key"] = proj.Key
	vars["permWorkflowName"] = w.Name
	vars["nodeID"] = fmt.Sprintf("%d", w.WorkflowData.Node.ID)

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
	assert.Len(t, models, 4, "")
}

func Test_getWorkflowHookModelHandler(t *testing.T) {
	api, _, _, end := newTestAPI(t)
	defer end()
	test.NoError(t, workflow.CreateBuiltinWorkflowHookModels(api.mustDB()))
	admin, passAdmin := assets.InsertAdminUser(t, api.mustDB())

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
	assert.Len(t, model.DefaultConfig, len(sdk.WebHookModel.DefaultConfig.Clone()))
}

func Test_putWorkflowHookModelHandlerAsAdminUser(t *testing.T) {
	api, _, _, end := newTestAPI(t)
	defer end()
	test.NoError(t, workflow.CreateBuiltinWorkflowHookModels(api.mustDB()))
	admin, passAdmin := assets.InsertAdminUser(t, api.mustDB())

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
	api, _, _, end := newTestAPI(t)
	defer end()
	test.NoError(t, workflow.CreateBuiltinWorkflowHookModels(api.mustDB()))

	u, pass := assets.InsertLambdaUser(t, api.mustDB())

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
