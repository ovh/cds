package main

import (
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"

	"github.com/ovh/cds/engine/api/auth"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/sdk"
)

func Test_getWorkflowsHandler(t *testing.T) {
	// Init database
	db := test.SetupPG(t)

	// Init router
	router = &Router{auth.TestLocalAuth(t), mux.NewRouter(), "/Test_getWorkflowsHandler"}
	router.init()
	// Init user
	u, pass := assets.InsertAdminUser(db)
	// Init project
	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, key, key, u)
	//Prepare request
	vars := map[string]string{
		"permProjectKey": proj.Key,
	}
	uri := router.getRoute("GET", getWorkflowsHandler, vars)
	test.NotEmpty(t, uri)
	req := assets.NewAuthentifiedRequest(t, u, pass, "GET", uri, vars)

	//Do the request
	w := httptest.NewRecorder()
	router.mux.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)
}

func Test_getWorkflowHandler(t *testing.T) {
	// Init database
	db := test.SetupPG(t)

	// Init router
	router = &Router{auth.TestLocalAuth(t), mux.NewRouter(), "/Test_getWorkflowHandler"}
	router.init()
	// Init user
	u, pass := assets.InsertAdminUser(db)
	// Init project
	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, key, key, u)
	//Prepare request
	vars := map[string]string{
		"permProjectKey": proj.Key,
		"workflowName":   "workflow1",
	}
	uri := router.getRoute("GET", getWorkflowHandler, vars)
	test.NotEmpty(t, uri)

	req := assets.NewAuthentifiedRequest(t, u, pass, "GET", uri, nil)
	//Do the request
	w := httptest.NewRecorder()
	router.mux.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)
}

func Test_postWorkflowHandlerWithoutRootShouldFail(t *testing.T) {
	// Init database
	db := test.SetupPG(t)
	// Init router
	router = &Router{auth.TestLocalAuth(t), mux.NewRouter(), "/Test_postWorkflowHandler"}
	router.init()
	// Init user
	u, pass := assets.InsertAdminUser(db)
	// Init project
	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, key, key, u)
	//Prepare request
	vars := map[string]string{
		"permProjectKey": proj.Key,
		"workflowName":   "workflow1",
	}
	uri := router.getRoute("POST", postWorkflowHandler, vars)
	test.NotEmpty(t, uri)

	var workflowResponse sdk.Workflow
	req := assets.NewAuthentifiedRequest(t, u, pass, "POST", uri, &workflowResponse)
	//Do the request
	w := httptest.NewRecorder()
	router.mux.ServeHTTP(w, req)
	assert.Equal(t, 400, w.Code)
	//assert.NotEqual(t, 0, workflowResponse.ID)
}

func Test_putWorkflowHandler(t *testing.T) {
	// Init database
	db := test.SetupPG(t)
	// Init router
	router = &Router{auth.TestLocalAuth(t), mux.NewRouter(), "/Test_putWorkflowHandler"}
	router.init()
	// Init user
	u, pass := assets.InsertAdminUser(db)
	// Init project
	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, key, key, u)
	//Prepare request
	vars := map[string]string{
		"permProjectKey": proj.Key,
		"workflowName":   "workflow1",
	}
	uri := router.getRoute("PUT", putWorkflowHandler, vars)
	test.NotEmpty(t, uri)

	req := assets.NewAuthentifiedRequest(t, u, pass, "PUT", uri, nil)
	//Do the request
	w := httptest.NewRecorder()
	router.mux.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)
}

func Test_deleteWorkflowHandler(t *testing.T) {
	// Init database
	db := test.SetupPG(t)
	// Init router
	router = &Router{auth.TestLocalAuth(t), mux.NewRouter(), "/Test_deleteWorkflowHandler"}
	router.init()
	// Init user
	u, pass := assets.InsertAdminUser(db)
	// Init project
	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, key, key, u)
	//Prepare request
	vars := map[string]string{
		"permProjectKey": proj.Key,
		"workflowName":   "workflow1",
	}
	uri := router.getRoute("DELETE", deleteWorkflowHandler, vars)
	test.NotEmpty(t, uri)

	req := assets.NewAuthentifiedRequest(t, u, pass, "DELETE", uri, nil)
	//Do the request
	w := httptest.NewRecorder()
	router.mux.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)
}
