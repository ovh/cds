package api

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ovh/cds/engine/api/environment"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/sdk"
)

func TestAddVariableInEnvironmentHandler(t *testing.T) {
	api, db, router, end := newTestAPI(t)
	defer end()

	//1. Create admin user
	u, pass := assets.InsertAdminUser(t, api.mustDB())

	//2. Create project
	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))
	test.NotNil(t, proj)

	//3. Create env
	env := sdk.Environment{
		ProjectID: proj.ID,
		Name:      "Prod",
	}
	if err := environment.InsertEnvironment(api.mustDB(), &env); err != nil {
		t.Fail()
		return
	}

	//4. Prepare the request
	addVarRequest := sdk.Variable{
		Name:  "foo",
		Value: "bar",
		Type:  sdk.StringVariable,
	}
	jsonBody, _ := json.Marshal(addVarRequest)
	body := bytes.NewBuffer(jsonBody)

	vars := map[string]string{
		"permProjectKey":  proj.Key,
		"environmentName": "Prod",
		"name":            addVarRequest.Name,
	}

	uri := router.GetRoute("POST", api.addVariableInEnvironmentHandler, vars)
	test.NotEmpty(t, uri)

	req, err := http.NewRequest("POST", uri, body)
	test.NoError(t, err)
	assets.AuthentifyRequest(t, req, u, pass)

	//4. Do the request
	w := httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	res, _ := ioutil.ReadAll(w.Body)
	v := &sdk.Variable{}
	assert.NoError(t, json.Unmarshal(res, &v))
	assert.NotEqual(t, v.ID, 0)

	envDb, err := environment.LoadEnvironmentByName(api.mustDB(), proj.Key, "Prod")
	if err != nil {
		t.Fail()
		return
	}
	assert.Equal(t, len(envDb.Variables), 1)
	assert.Equal(t, envDb.Variables[0].Name, "foo")
}

func TestUpdateVariableInEnvironmentHandler(t *testing.T) {
	api, db, router, end := newTestAPI(t)
	defer end()

	//1. Create admin user
	u, pass := assets.InsertAdminUser(t, api.mustDB())

	//2. Create project
	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))
	test.NotNil(t, proj)

	//3. Create env
	env := sdk.Environment{
		ProjectID: proj.ID,
		Name:      "Prod",
	}
	if err := environment.InsertEnvironment(api.mustDB(), &env); err != nil {
		t.Fail()
		return
	}

	//4. add a variable
	v := sdk.Variable{
		Name:  "foo",
		Value: "bar",
		Type:  sdk.StringVariable,
	}
	if err := environment.InsertVariable(api.mustDB(), env.ID, &v, u); err != nil {
		t.Fail()
		return
	}

	//4. Prepare the request
	v.Value = "new bar"

	jsonBody, _ := json.Marshal(v)
	body := bytes.NewBuffer(jsonBody)

	vars := map[string]string{
		"permProjectKey":  proj.Key,
		"environmentName": "Prod",
		"name":            v.Name,
	}

	uri := router.GetRoute("PUT", api.updateVariableInEnvironmentHandler, vars)
	test.NotEmpty(t, uri)

	req, err := http.NewRequest("PUT", uri, body)
	test.NoError(t, err)
	assets.AuthentifyRequest(t, req, u, pass)

	//5. Do the request
	w := httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	res, _ := ioutil.ReadAll(w.Body)
	vUpdated := sdk.Variable{}
	assert.NoError(t, json.Unmarshal(res, &vUpdated))
	assert.Equal(t, vUpdated.Value, "new bar")

	envDb, err := environment.LoadEnvironmentByName(api.mustDB(), proj.Key, "Prod")
	if err != nil {
		t.Fail()
		return
	}
	assert.Equal(t, len(envDb.Variables), 1)
	assert.Equal(t, envDb.Variables[0].Value, "new bar")
}

func TestDeleteVariableFromEnvironmentHandler(t *testing.T) {
	api, db, router, end := newTestAPI(t)
	defer end()

	//1. Create admin user
	u, pass := assets.InsertAdminUser(t, api.mustDB())

	//2. Create project
	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))
	test.NotNil(t, proj)

	//3. Create env
	env := sdk.Environment{
		ProjectID: proj.ID,
		Name:      "Prod",
	}
	if err := environment.InsertEnvironment(api.mustDB(), &env); err != nil {
		t.Fail()
		return
	}

	//4. add a variable
	v := sdk.Variable{
		Name:  "foo",
		Value: "bar",
		Type:  sdk.StringVariable,
	}
	if err := environment.InsertVariable(api.mustDB(), env.ID, &v, u); err != nil {
		t.Fail()
		return
	}

	vars := map[string]string{
		"permProjectKey":  proj.Key,
		"environmentName": "Prod",
		"name":            v.Name,
	}

	uri := router.GetRoute("DELETE", api.deleteVariableFromEnvironmentHandler, vars)
	test.NotEmpty(t, uri)

	req, err := http.NewRequest("DELETE", uri, nil)
	test.NoError(t, err)
	assets.AuthentifyRequest(t, req, u, pass)

	//5. Do the request
	w := httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)

	envDb, err := environment.LoadEnvironmentByName(api.mustDB(), proj.Key, "Prod")
	if err != nil {
		t.Fail()
		return
	}
	assert.Equal(t, len(envDb.Variables), 0)
}

func TestGetVariablesInEnvironmentHandler(t *testing.T) {
	api, db, router, end := newTestAPI(t)
	defer end()

	//1. Create admin user
	u, pass := assets.InsertAdminUser(t, api.mustDB())

	//2. Create project
	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))
	test.NotNil(t, proj)

	//3. Create env
	env := sdk.Environment{
		ProjectID: proj.ID,
		Name:      "Prod",
	}
	if err := environment.InsertEnvironment(api.mustDB(), &env); err != nil {
		t.Fail()
		return
	}

	//4. add a variable
	v := sdk.Variable{
		Name:  "foo",
		Value: "bar",
		Type:  sdk.StringVariable,
	}
	if err := environment.InsertVariable(api.mustDB(), env.ID, &v, u); err != nil {
		t.Fail()
		return
	}

	vars := map[string]string{
		"permProjectKey":  proj.Key,
		"environmentName": "Prod",
		"name":            v.Name,
	}

	uri := router.GetRoute("GET", api.getVariablesInEnvironmentHandler, vars)
	test.NotEmpty(t, uri)

	req, _ := http.NewRequest("GET", uri, nil)
	assets.AuthentifyRequest(t, req, u, pass)

	//5. Do the request
	w := httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	res, _ := ioutil.ReadAll(w.Body)
	varsResult := []sdk.Variable{}
	json.Unmarshal(res, &varsResult)
	assert.Equal(t, len(varsResult), 1)
	assert.Equal(t, varsResult[0].Name, "foo")
}

func Test_getVariableAuditInEnvironmentHandler(t *testing.T) {
	api, db, router, end := newTestAPI(t)
	defer end()

	//Create admin user
	u, pass := assets.InsertAdminUser(t, api.mustDB())

	//Insert Project
	pkey := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, api.Cache, pkey, pkey)

	// Insert env
	e := &sdk.Environment{
		Name:      "Production",
		ProjectID: proj.ID,
	}
	if err := environment.InsertEnvironment(api.mustDB(), e); err != nil {
		t.Fatal(err)
	}

	// Add variable
	v := sdk.Variable{
		Name:  "foo",
		Type:  "string",
		Value: "bar",
	}
	if err := environment.InsertVariable(api.mustDB(), e.ID, &v, u); err != nil {
		t.Fatal(err)
	}

	vars := map[string]string{
		"permProjectKey":  proj.Key,
		"environmentName": e.Name,
		"name":            "foo",
	}

	uri := router.GetRoute("GET", api.getVariableAuditInEnvironmentHandler, vars)
	req, err := http.NewRequest("GET", uri, nil)
	test.NoError(t, err)
	assets.AuthentifyRequest(t, req, u, pass)

	// Do the request
	w := httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)

	var audits []sdk.EnvironmentVariableAudit
	test.NoError(t, json.Unmarshal(w.Body.Bytes(), &audits))
	assert.Equal(t, len(audits), 1)

	assert.Nil(t, audits[0].VariableBefore)
	assert.Equal(t, "foo", audits[0].VariableAfter.Name)
}
