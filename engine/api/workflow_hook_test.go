package main

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"

	"github.com/ovh/cds/engine/api/auth"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/sdk"
)

func Test_getWorkflowHookModelsHandlerAsLambdaUser(t *testing.T) {
	db := test.SetupPG(t)
	test.NoError(t, workflow.CreateBuiltinWorkflowHookModels(db))
	user, passUser := assets.InsertLambdaUser(db)

	// Init router
	router = newRouter(auth.TestLocalAuth(t), mux.NewRouter(), "/Test_getWorkflowHookModelsHandlerAsLambdaUser")
	router.init()
	//Prepare request
	vars := map[string]string{}
	uri := router.getRoute("GET", getWorkflowHookModelsHandler, vars)
	test.NotEmpty(t, uri)
	req := assets.NewAuthentifiedRequest(t, user, passUser, "GET", uri, nil)

	//Do the request
	rec := httptest.NewRecorder()
	router.mux.ServeHTTP(rec, req)
	assert.Equal(t, 200, rec.Code)

	//Check result
	models := []sdk.WorkflowHookModel{}
	test.NoError(t, json.Unmarshal(rec.Body.Bytes(), &models))
	assert.Len(t, models, 3, "")
}

func Test_getWorkflowHookModelsHandlerAsAdminUser(t *testing.T) {
	db := test.SetupPG(t)
	test.NoError(t, workflow.CreateBuiltinWorkflowHookModels(db))
	admin, passAdmin := assets.InsertAdminUser(db)

	// Init router
	router = newRouter(auth.TestLocalAuth(t), mux.NewRouter(), "/Test_getWorkflowHookModelsHandlerAsAdminUser")
	router.init()
	//Prepare request
	vars := map[string]string{}
	uri := router.getRoute("GET", getWorkflowHookModelsHandler, vars)
	test.NotEmpty(t, uri)
	req := assets.NewAuthentifiedRequest(t, admin, passAdmin, "GET", uri, nil)

	//Do the request
	rec := httptest.NewRecorder()
	router.mux.ServeHTTP(rec, req)
	assert.Equal(t, 200, rec.Code)

	//Check result
	models := []sdk.WorkflowHookModel{}
	test.NoError(t, json.Unmarshal(rec.Body.Bytes(), &models))
	assert.Len(t, models, 3, "")
}

func Test_getWorkflowHookModelHandler(t *testing.T) {
	db := test.SetupPG(t)
	test.NoError(t, workflow.CreateBuiltinWorkflowHookModels(db))
	admin, passAdmin := assets.InsertAdminUser(db)

	// Init router
	router = newRouter(auth.TestLocalAuth(t), mux.NewRouter(), "/Test_getWorkflowHookModelHandler")
	router.init()
	//Prepare request
	vars := map[string]string{
		"model": workflow.WebHookModel.Name,
	}
	uri := router.getRoute("GET", getWorkflowHookModelHandler, vars)
	test.NotEmpty(t, uri)
	req := assets.NewAuthentifiedRequest(t, admin, passAdmin, "GET", uri, nil)

	//Do the request
	rec := httptest.NewRecorder()
	router.mux.ServeHTTP(rec, req)

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
	db := test.SetupPG(t)
	test.NoError(t, workflow.CreateBuiltinWorkflowHookModels(db))
	admin, passAdmin := assets.InsertAdminUser(db)

	// Init router
	router = newRouter(auth.TestLocalAuth(t), mux.NewRouter(), "/Test_putWorkflowHookModelHandlerAsAdminUser")
	router.init()
	//Prepare request
	vars := map[string]string{
		"model": workflow.WebHookModel.Name,
	}
	uri := router.getRoute("GET", getWorkflowHookModelHandler, vars)
	test.NotEmpty(t, uri)
	req := assets.NewAuthentifiedRequest(t, admin, passAdmin, "GET", uri, nil)

	//Do the request
	rec := httptest.NewRecorder()
	router.mux.ServeHTTP(rec, req)

	//Check result
	model := sdk.WorkflowHookModel{}
	test.NoError(t, json.Unmarshal(rec.Body.Bytes(), &model))
	assert.Equal(t, workflow.WebHookModel.Name, model.Name)

	//Now update it
	model.Disabled = false

	//Prepare the request
	uri = router.getRoute("PUT", putWorkflowHookModelHandler, vars)
	test.NotEmpty(t, uri)
	req = assets.NewAuthentifiedRequest(t, admin, passAdmin, "PUT", uri, model)

	//Do the request
	rec = httptest.NewRecorder()
	router.mux.ServeHTTP(rec, req)

	//Check result
	model = sdk.WorkflowHookModel{}
	test.NoError(t, json.Unmarshal(rec.Body.Bytes(), &model))
	assert.Equal(t, workflow.WebHookModel.Name, model.Name)
	assert.Equal(t, false, model.Disabled)

}

func Test_putWorkflowHookModelHandlerAsLambdaUser(t *testing.T) {
	db := test.SetupPG(t)
	test.NoError(t, workflow.CreateBuiltinWorkflowHookModels(db))

	u, pass := assets.InsertLambdaUser(db)

	// Init router
	router = newRouter(auth.TestLocalAuth(t), mux.NewRouter(), "/Test_putWorkflowHookModelHandlerAsAdminUser")
	router.init()
	//Prepare request
	vars := map[string]string{
		"model": workflow.WebHookModel.Name,
	}
	uri := router.getRoute("GET", getWorkflowHookModelHandler, vars)
	test.NotEmpty(t, uri)
	req := assets.NewAuthentifiedRequest(t, u, pass, "GET", uri, nil)

	//Do the request
	rec := httptest.NewRecorder()
	router.mux.ServeHTTP(rec, req)

	//Check result
	model := sdk.WorkflowHookModel{}
	test.NoError(t, json.Unmarshal(rec.Body.Bytes(), &model))
	assert.Equal(t, workflow.WebHookModel.Name, model.Name)

	//Now update it
	model.Disabled = false

	//Prepare the request
	uri = router.getRoute("PUT", putWorkflowHookModelHandler, vars)
	test.NotEmpty(t, uri)
	req = assets.NewAuthentifiedRequest(t, u, pass, "PUT", uri, model)

	//Do the request
	rec = httptest.NewRecorder()
	router.mux.ServeHTTP(rec, req)

	assert.Equal(t, 403, rec.Code)

}
