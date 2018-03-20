package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/sdk"
)

func Test_getVariableAuditInProjectHandler(t *testing.T) {
	api, db, router := newTestAPI(t)

	//Create admin user
	u, pass := assets.InsertAdminUser(api.mustDB())

	//Insert Project
	pkey := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, api.Cache, pkey, pkey, u)

	// Add variable
	v := sdk.Variable{
		Name:  "foo",
		Type:  "string",
		Value: "bar",
	}
	if err := project.InsertVariable(api.mustDB(), proj, &v, u); err != nil {
		t.Fatal(err)
	}

	vars := map[string]string{
		"permProjectKey": proj.Key,
		"name":           "foo",
	}

	uri := router.GetRoute("GET", api.getVariableAuditInProjectHandler, vars)
	req, err := http.NewRequest("GET", uri, nil)
	test.NoError(t, err)
	assets.AuthentifyRequest(t, req, u, pass)

	// Do the request
	w := httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)

	var audits []sdk.ProjectVariableAudit
	test.NoError(t, json.Unmarshal(w.Body.Bytes(), &audits))

	assert.Nil(t, audits[0].VariableBefore)
	assert.Equal(t, audits[0].VariableAfter.Name, "foo")
}

func Test_postEncryptVariableHandler(t *testing.T) {
	api, db, router := newTestAPI(t)

	//Create admin user
	u, pass := assets.InsertAdminUser(api.mustDB())

	//Insert Project
	pkey := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, api.Cache, pkey, pkey, u)

	vars := map[string]string{
		"permProjectKey": proj.Key,
	}

	// Add variable
	v := &sdk.Variable{
		Name:  "foo",
		Type:  sdk.SecretVariable,
		Value: "bar",
	}

	uri := router.GetRoute("POST", api.postEncryptVariableHandler, vars)
	req := assets.NewAuthentifiedRequest(t, u, pass, "POST", uri, v)

	//Do the request
	rec := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	assert.Equal(t, 200, rec.Code)

	//Check result
	test.NoError(t, json.Unmarshal(rec.Body.Bytes(), v))

	decrypt, err := project.DecryptWithBuiltinKey(db, proj.ID, v.Value)
	test.NoError(t, err)

	assert.Equal(t, "bar", decrypt)
}
