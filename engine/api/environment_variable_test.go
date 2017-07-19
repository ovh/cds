package main

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/loopfz/gadgeto/iffy"
	"github.com/stretchr/testify/assert"

	"github.com/ovh/cds/engine/api/auth"
	"github.com/ovh/cds/engine/api/environment"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/sdk"
)

func TestAddVariableInEnvironmentHandler(t *testing.T) {
	db := test.SetupPG(t)

	router = newRouter(auth.TestLocalAuth(t), mux.NewRouter(), "/TestAddVariableInEnvironmentHandler")
	router.init()

	//1. Create admin user
	u, pass := assets.InsertAdminUser(db)

	//2. Create project
	proj := assets.InsertTestProject(t, db, sdk.RandomString(10), sdk.RandomString(10), u)
	test.NotNil(t, proj)

	//3. Create env
	env := sdk.Environment{
		ProjectID: proj.ID,
		Name:      "Prod",
	}
	if err := environment.InsertEnvironment(db, &env); err != nil {
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
		"key": proj.Key,
		"permEnvironmentName": "Prod",
		"name":                addVarRequest.Name,
	}

	uri := router.getRoute("POST", addVariableInEnvironmentHandler, vars)
	test.NotEmpty(t, uri)

	req, err := http.NewRequest("POST", uri, body)
	assets.AuthentifyRequest(t, req, u, pass)

	//4. Do the request
	w := httptest.NewRecorder()
	router.mux.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	res, _ := ioutil.ReadAll(w.Body)
	projectResult := &sdk.Project{}
	json.Unmarshal(res, &projectResult)
	assert.Equal(t, len(projectResult.Environments), 1)
	assert.Equal(t, len(projectResult.Environments[0].Variable), 1)

	envDb, err := environment.LoadEnvironmentByName(db, proj.Key, "Prod")
	if err != nil {
		t.Fail()
		return
	}
	assert.Equal(t, len(envDb.Variable), 1)
	assert.Equal(t, envDb.Variable[0].Name, "foo")
}

func TestUpdateVariableInEnvironmentHandler(t *testing.T) {
	db := test.SetupPG(t)

	router = newRouter(auth.TestLocalAuth(t), mux.NewRouter(), "/TestUpdateVariableInEnvironmentHandler")
	router.init()

	//1. Create admin user
	u, pass := assets.InsertAdminUser(db)

	//2. Create project
	proj := assets.InsertTestProject(t, db, sdk.RandomString(10), sdk.RandomString(10), u)
	test.NotNil(t, proj)

	//3. Create env
	env := sdk.Environment{
		ProjectID: proj.ID,
		Name:      "Prod",
	}
	if err := environment.InsertEnvironment(db, &env); err != nil {
		t.Fail()
		return
	}

	//4. add a variable
	v := sdk.Variable{
		Name:  "foo",
		Value: "bar",
		Type:  sdk.StringVariable,
	}
	if err := environment.InsertVariable(db, env.ID, &v, u); err != nil {
		t.Fail()
		return
	}

	//4. Prepare the request
	v.Value = "new bar"

	jsonBody, _ := json.Marshal(v)
	body := bytes.NewBuffer(jsonBody)

	vars := map[string]string{
		"key": proj.Key,
		"permEnvironmentName": "Prod",
		"name":                v.Name,
	}

	uri := router.getRoute("PUT", updateVariableInEnvironmentHandler, vars)
	test.NotEmpty(t, uri)

	req, err := http.NewRequest("PUT", uri, body)
	assets.AuthentifyRequest(t, req, u, pass)

	//5. Do the request
	w := httptest.NewRecorder()
	router.mux.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	res, _ := ioutil.ReadAll(w.Body)
	projectResult := &sdk.Project{}
	json.Unmarshal(res, &projectResult)
	assert.Equal(t, len(projectResult.Environments), 1)
	assert.Equal(t, len(projectResult.Environments[0].Variable), 1)

	envDb, err := environment.LoadEnvironmentByName(db, proj.Key, "Prod")
	if err != nil {
		t.Fail()
		return
	}
	assert.Equal(t, len(envDb.Variable), 1)
	assert.Equal(t, envDb.Variable[0].Value, "new bar")
}

func TestDeleteVariableFromEnvironmentHandler(t *testing.T) {
	db := test.SetupPG(t)

	router = newRouter(auth.TestLocalAuth(t), mux.NewRouter(), "/TestDeleteVariableFromEnvironmentHandler")
	router.init()

	//1. Create admin user
	u, pass := assets.InsertAdminUser(db)

	//2. Create project
	proj := assets.InsertTestProject(t, db, sdk.RandomString(10), sdk.RandomString(10), u)
	test.NotNil(t, proj)

	//3. Create env
	env := sdk.Environment{
		ProjectID: proj.ID,
		Name:      "Prod",
	}
	if err := environment.InsertEnvironment(db, &env); err != nil {
		t.Fail()
		return
	}

	//4. add a variable
	v := sdk.Variable{
		Name:  "foo",
		Value: "bar",
		Type:  sdk.StringVariable,
	}
	if err := environment.InsertVariable(db, env.ID, &v, u); err != nil {
		t.Fail()
		return
	}

	vars := map[string]string{
		"key": proj.Key,
		"permEnvironmentName": "Prod",
		"name":                v.Name,
	}

	uri := router.getRoute("DELETE", deleteVariableFromEnvironmentHandler, vars)
	test.NotEmpty(t, uri)

	req, err := http.NewRequest("DELETE", uri, nil)
	assets.AuthentifyRequest(t, req, u, pass)

	//5. Do the request
	w := httptest.NewRecorder()
	router.mux.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	res, _ := ioutil.ReadAll(w.Body)
	projectResult := &sdk.Project{}
	json.Unmarshal(res, &projectResult)
	assert.Equal(t, len(projectResult.Environments), 1)
	assert.Equal(t, len(projectResult.Environments[0].Variable), 0)

	envDb, err := environment.LoadEnvironmentByName(db, proj.Key, "Prod")
	if err != nil {
		t.Fail()
		return
	}
	assert.Equal(t, len(envDb.Variable), 0)
}

func TestGetVariablesInEnvironmentHandler(t *testing.T) {
	db := test.SetupPG(t)

	router = newRouter(auth.TestLocalAuth(t), mux.NewRouter(), "/TestGetVariablesInEnvironmentHandler")
	router.init()

	//1. Create admin user
	u, pass := assets.InsertAdminUser(db)

	//2. Create project
	proj := assets.InsertTestProject(t, db, sdk.RandomString(10), sdk.RandomString(10), u)
	test.NotNil(t, proj)

	//3. Create env
	env := sdk.Environment{
		ProjectID: proj.ID,
		Name:      "Prod",
	}
	if err := environment.InsertEnvironment(db, &env); err != nil {
		t.Fail()
		return
	}

	//4. add a variable
	v := sdk.Variable{
		Name:  "foo",
		Value: "bar",
		Type:  sdk.StringVariable,
	}
	if err := environment.InsertVariable(db, env.ID, &v, u); err != nil {
		t.Fail()
		return
	}

	vars := map[string]string{
		"key": proj.Key,
		"permEnvironmentName": "Prod",
		"name":                v.Name,
	}

	uri := router.getRoute("GET", getVariablesInEnvironmentHandler, vars)
	test.NotEmpty(t, uri)

	req, _ := http.NewRequest("GET", uri, nil)
	assets.AuthentifyRequest(t, req, u, pass)

	//5. Do the request
	w := httptest.NewRecorder()
	router.mux.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	res, _ := ioutil.ReadAll(w.Body)
	varsResult := []sdk.Variable{}
	json.Unmarshal(res, &varsResult)
	assert.Equal(t, len(varsResult), 1)
	assert.Equal(t, varsResult[0].Name, "foo")
}

func Test_getVariableAuditInEnvironmentHandler(t *testing.T) {
	db := test.SetupPG(t)

	router = newRouter(auth.TestLocalAuth(t), mux.NewRouter(), "/Test_getVariableAuditInEnvironmentHandler")
	router.init()

	//Create admin user
	u, pass := assets.InsertAdminUser(db)

	//Create a fancy httptester
	tester := iffy.NewTester(t, router.mux)

	//Insert Project
	pkey := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, pkey, pkey, u)

	// Insert env
	e := &sdk.Environment{
		Name:      "Production",
		ProjectID: proj.ID,
	}
	if err := environment.InsertEnvironment(db, e); err != nil {
		t.Fatal(err)
	}

	// Add variable
	v := sdk.Variable{
		Name:  "foo",
		Type:  "string",
		Value: "bar",
	}
	if err := environment.InsertVariable(db, e.ID, &v, u); err != nil {
		t.Fatal(err)
	}

	vars := map[string]string{
		"key": proj.Key,
		"permEnvironmentName": e.Name,
		"name":                "foo",
	}

	route := router.getRoute("GET", getVariableAuditInEnvironmentHandler, vars)
	headers := assets.AuthHeaders(t, u, pass)

	var audits []sdk.EnvironmentVariableAudit
	tester.AddCall("Test_getVariableAuditInEnvironmentHandler", "GET", route, nil).Headers(headers).Checkers(iffy.ExpectStatus(200), iffy.ExpectListLength(1), iffy.DumpResponse(t), iffy.UnmarshalResponse(&audits))
	tester.Run()

	assert.Nil(t, audits[0].VariableBefore)
	assert.Equal(t, "foo", audits[0].VariableAfter.Name)
}
