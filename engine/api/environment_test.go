package main

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"

	"github.com/ovh/cds/engine/api/environment"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/sdk"
)

func TestAddEnvironmentHandler(t *testing.T) {
	db := test.SetupPG(t)

	router = &Router{test.LocalAuth(t), mux.NewRouter(), "/TestAddEnvironmentHandler"}
	router.init()

	//1. Create admin user
	u, pass := test.InsertAdminUser(t, db)

	//2. Create project
	proj := test.InsertTestProject(t, db, test.RandomString(t, 10), test.RandomString(t, 10))
	test.NotNil(t, proj)

	//3. Prepare the request
	addEnvRequest := sdk.Environment{
		Name: "Production",
	}
	jsonBody, _ := json.Marshal(addEnvRequest)
	body := bytes.NewBuffer(jsonBody)

	vars := map[string]string{
		"permProjectKey": proj.Key,
	}

	uri := router.getRoute("POST", addEnvironmentHandler, vars)
	test.NotEmpty(t, uri)

	req, err := http.NewRequest("POST", uri, body)
	test.AuthentifyRequest(t, req, u, pass)

	//4. Do the request
	w := httptest.NewRecorder()
	router.mux.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	res, _ := ioutil.ReadAll(w.Body)
	projectResult := &sdk.Project{}
	json.Unmarshal(res, &projectResult)
	assert.Equal(t, len(projectResult.Environments), 1)

	env, err := environment.LoadEnvironmentByName(db, proj.Key, "Production")
	if err != nil {
		t.Fail()
		return
	}
	assert.Equal(t, env.Name, "Production")
}

func TestUpdateEnvironmentHandler(t *testing.T) {
	db := test.SetupPG(t)

	router = &Router{test.LocalAuth(t), mux.NewRouter(), "/TestUpdateEnvironmentHandler"}
	router.init()

	//1. Create admin user
	u, pass := test.InsertAdminUser(t, db)

	//2. Create project
	proj := test.InsertTestProject(t, db, test.RandomString(t, 10), test.RandomString(t, 10))
	test.NotNil(t, proj)

	//3. Create env
	env := sdk.Environment{
		ProjectID: proj.ID,
		Name:      "Preproduction",
	}
	if err := environment.InsertEnvironment(db, &env); err != nil {
		t.Fail()
		return
	}

	//4. Prepare the request
	env.Name = "Production"

	jsonBody, _ := json.Marshal(env)
	body := bytes.NewBuffer(jsonBody)

	vars := map[string]string{
		"key": proj.Key,
		"permEnvironmentName": "Preproduction",
	}

	uri := router.getRoute("PUT", updateEnvironmentHandler, vars)
	test.NotEmpty(t, uri)

	req, err := http.NewRequest("PUT", uri, body)
	test.AuthentifyRequest(t, req, u, pass)

	//5. Do the request
	w := httptest.NewRecorder()
	router.mux.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	res, _ := ioutil.ReadAll(w.Body)
	projectResult := &sdk.Project{}
	json.Unmarshal(res, &projectResult)
	assert.Equal(t, len(projectResult.Environments), 1)

	envDb, err := environment.LoadEnvironmentByName(db, proj.Key, "Production")
	if err != nil {
		t.Fail()
		return
	}
	assert.Equal(t, envDb.Name, "Production")
}

func TestDeleteEnvironmentHandler(t *testing.T) {
	db := test.SetupPG(t)

	router = &Router{test.LocalAuth(t), mux.NewRouter(), "/TestDeleteEnvironmentHandler"}
	router.init()

	//1. Create admin user
	u, pass := test.InsertAdminUser(t, db)

	//2. Create project
	proj := test.InsertTestProject(t, db, test.RandomString(t, 10), test.RandomString(t, 10))
	test.NotNil(t, proj)

	//3. Create env
	env := sdk.Environment{
		ProjectID: proj.ID,
		Name:      "Preproduction",
	}
	if err := environment.InsertEnvironment(db, &env); err != nil {
		t.Fail()
		return
	}

	vars := map[string]string{
		"key": proj.Key,
		"permEnvironmentName": "Preproduction",
	}

	uri := router.getRoute("DELETE", deleteEnvironmentHandler, vars)
	test.NotEmpty(t, uri)

	req, err := http.NewRequest("DELETE", uri, nil)
	test.AuthentifyRequest(t, req, u, pass)

	//4. Do the request
	w := httptest.NewRecorder()
	router.mux.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	res, _ := ioutil.ReadAll(w.Body)
	projectResult := &sdk.Project{}
	json.Unmarshal(res, &projectResult)
	assert.Equal(t, len(projectResult.Environments), 0)

	_, err = environment.LoadEnvironmentByName(db, proj.Key, "Preproduction")
	if err == sdk.ErrNoEnvironment {
		return
	}
	t.Fail()
}

func TestGetEnvironmentsHandler(t *testing.T) {
	db := test.SetupPG(t)

	router = &Router{test.LocalAuth(t), mux.NewRouter(), "/TestDeleteEnvironmentHandler"}
	router.init()

	//1. Create admin user
	u, pass := test.InsertAdminUser(t, db)

	//2. Create project
	proj := test.InsertTestProject(t, db, test.RandomString(t, 10), test.RandomString(t, 10))
	test.NotNil(t, proj)

	//3. Create env
	env := sdk.Environment{
		ProjectID: proj.ID,
		Name:      "Preproduction",
	}
	if err := environment.InsertEnvironment(db, &env); err != nil {
		t.Fail()
		return
	}

	vars := map[string]string{
		"permProjectKey": proj.Key,
	}

	uri := router.getRoute("GET", getEnvironmentsHandler, vars)
	test.NotEmpty(t, uri)

	req, _ := http.NewRequest("GET", uri, nil)
	test.AuthentifyRequest(t, req, u, pass)

	//4. Do the request
	w := httptest.NewRecorder()
	router.mux.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	res, _ := ioutil.ReadAll(w.Body)
	envsResults := []sdk.Environment{}
	json.Unmarshal(res, &envsResults)
	assert.Equal(t, len(envsResults), 1)
}

func TestGetEnvironmentHandler(t *testing.T) {
	db := test.SetupPG(t)

	router = &Router{test.LocalAuth(t), mux.NewRouter(), "/TestDeleteEnvironmentHandler"}
	router.init()

	//1. Create admin user
	u, pass := test.InsertAdminUser(t, db)

	//2. Create project
	proj := test.InsertTestProject(t, db, test.RandomString(t, 10), test.RandomString(t, 10))
	test.NotNil(t, proj)

	//3. Create env
	env := sdk.Environment{
		ProjectID: proj.ID,
		Name:      "Preproduction",
	}
	if err := environment.InsertEnvironment(db, &env); err != nil {
		t.Fail()
		return
	}

	vars := map[string]string{
		"key": proj.Key,
		"permEnvironmentName": env.Name,
	}

	uri := router.getRoute("GET", getEnvironmentHandler, vars)
	test.NotEmpty(t, uri)

	req, _ := http.NewRequest("GET", uri, nil)
	test.AuthentifyRequest(t, req, u, pass)

	//4. Do the request
	w := httptest.NewRecorder()
	router.mux.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	res, _ := ioutil.ReadAll(w.Body)
	envResults := sdk.Environment{}
	json.Unmarshal(res, &envResults)
	assert.Equal(t, envResults.Name, "Preproduction")
}

func Test_cloneEnvironmentHandler(t *testing.T) {
	db := test.SetupPG(t)

	router = &Router{test.LocalAuth(t), mux.NewRouter(), "/Test_cloneEnvironmentHandler"}
	router.init()

	//1. Create admin user
	u, pass := test.InsertAdminUser(t, db)

	//2. Create project
	proj := test.InsertTestProject(t, db, test.RandomString(t, 10), test.RandomString(t, 10))
	test.NotNil(t, proj)

	//3. Create env
	env := sdk.Environment{
		ProjectID: proj.ID,
		Name:      "Preproduction",
	}
	if err := environment.InsertEnvironment(db, &env); err != nil {
		t.Fail()
		return
	}

	vars := map[string]string{
		"key": proj.Key,
		"permEnvironmentName": env.Name,
	}

	uri := router.getRoute("GET", getEnvironmentHandler, vars)
}
