package api

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/sdk"
)

func Test_getWorkflowHookModelsHandlerAsLambdaUser(t *testing.T) {
	api, _, _ := newTestAPI(t)
	test.NoError(t, workflow.CreateBuiltinWorkflowHookModels(api.mustDB()))
	user, passUser := assets.InsertLambdaUser(api.mustDB())

	//Prepare request
	vars := map[string]string{}
	uri := api.Router.GetRoute("GET", api.getWorkflowHookModelsHandler, vars)
	test.NotEmpty(t, uri)
	req := assets.NewAuthentifiedRequest(t, user, passUser, "GET", uri, nil)

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
	api, _, _ := newTestAPI(t)
	test.NoError(t, workflow.CreateBuiltinWorkflowHookModels(api.mustDB()))
	admin, passAdmin := assets.InsertAdminUser(api.mustDB())

	//Prepare request
	vars := map[string]string{}
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
	api, _, _ := newTestAPI(t)
	test.NoError(t, workflow.CreateBuiltinWorkflowHookModels(api.mustDB()))
	admin, passAdmin := assets.InsertAdminUser(api.mustDB())

	//Prepare request
	vars := map[string]string{
		"model": workflow.WebHookModel.Name,
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
	assert.Equal(t, workflow.WebHookModel.Name, model.Name)
	assert.Equal(t, workflow.WebHookModel.Command, model.Command)
	assert.Equal(t, workflow.WebHookModel.Description, model.Description)
	assert.Equal(t, workflow.WebHookModel.Disabled, model.Disabled)
	assert.Len(t, model.DefaultConfig, len(workflow.WebHookModel.DefaultConfig))
}

func Test_putWorkflowHookModelHandlerAsAdminUser(t *testing.T) {
	api, _, _ := newTestAPI(t)
	test.NoError(t, workflow.CreateBuiltinWorkflowHookModels(api.mustDB()))
	admin, passAdmin := assets.InsertAdminUser(api.mustDB())

	//Prepare request
	vars := map[string]string{
		"model": workflow.WebHookModel.Name,
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
	assert.Equal(t, workflow.WebHookModel.Name, model.Name)

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
	assert.Equal(t, workflow.WebHookModel.Name, model.Name)
	assert.Equal(t, false, model.Disabled)

}

func Test_putWorkflowHookModelHandlerAsLambdaUser(t *testing.T) {
	api, _, _ := newTestAPI(t)
	test.NoError(t, workflow.CreateBuiltinWorkflowHookModels(api.mustDB()))

	u, pass := assets.InsertLambdaUser(api.mustDB())

	//Prepare request
	vars := map[string]string{
		"model": workflow.WebHookModel.Name,
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
	assert.Equal(t, workflow.WebHookModel.Name, model.Name)

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
