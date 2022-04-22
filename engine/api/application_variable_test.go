package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/sdk"
)

func Test_getVariableAuditInApplicationHandler(t *testing.T) {
	api, db, router := newTestAPI(t)

	//Create admin user
	u, pass := assets.InsertAdminUser(t, db)

	//Insert Project
	pkey := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, api.Cache, pkey, pkey)

	app := &sdk.Application{
		Name: sdk.RandomString(10),
	}
	require.NoError(t, application.Insert(db, *proj, app))

	// Add variable
	v := sdk.ApplicationVariable{
		Name:  "foo",
		Type:  "string",
		Value: "bar",
	}
	if err := application.InsertVariable(db, app.ID, &v, u); err != nil {
		t.Fatal(err)
	}

	vars := map[string]string{
		"permProjectKey":  proj.Key,
		"applicationName": app.Name,
		"name":            "foo",
	}

	uri := router.GetRoute("GET", api.getVariableAuditInApplicationHandler, vars)

	req, err := http.NewRequest("GET", uri, nil)
	require.NoError(t, err)
	assets.AuthentifyRequest(t, req, u, pass)

	// Do the request
	w := httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)

	var audits []sdk.ApplicationVariableAudit
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &audits))
	assert.Equal(t, len(audits), 1)

	assert.Nil(t, audits[0].VariableBefore)
	assert.Equal(t, audits[0].VariableAfter.Name, "foo")
}
