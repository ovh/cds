package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/sdk"
)

func Test_getVariableAuditInApplicationHandler(t *testing.T) {
	api, db, router := newTestAPI(t)

	//Create admin user
	u, pass := assets.InsertAdminUser(api.mustDB())

	//Insert Project
	pkey := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, api.Cache, pkey, pkey, u)

	app := &sdk.Application{
		Name: sdk.RandomString(10),
	}
	if err := application.Insert(api.mustDB(), api.Cache, proj, app, u); err != nil {
		t.Fatal(err)
	}

	// Add variable
	v := sdk.Variable{
		Name:  "foo",
		Type:  "string",
		Value: "bar",
	}
	if err := application.InsertVariable(api.mustDB(), api.Cache, app, v, u); err != nil {
		t.Fatal(err)
	}

	vars := map[string]string{
		"key": proj.Key,
		"permApplicationName": app.Name,
		"name":                "foo",
	}

	uri := router.GetRoute("GET", api.getVariableAuditInApplicationHandler, vars)

	req, err := http.NewRequest("GET", uri, nil)
	test.NoError(t, err)
	assets.AuthentifyRequest(t, req, u, pass)

	// Do the request
	w := httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)

	var audits []sdk.ApplicationVariableAudit
	test.NoError(t, json.Unmarshal(w.Body.Bytes(), &audits))
	assert.Equal(t, len(audits), 1)

	assert.Nil(t, audits[0].VariableBefore)
	assert.Equal(t, audits[0].VariableAfter.Name, "foo")
}
