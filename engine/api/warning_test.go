package api

import (
	"encoding/json"
	"github.com/ovh/cds/engine/api/bootstrap"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/engine/api/warning"
	"github.com/ovh/cds/sdk"
	"github.com/stretchr/testify/assert"
	"net/http/httptest"
	"testing"
	"time"
)

func Test_getWarningsHandler(t *testing.T) {
	api, db, _ := newTestAPI(t, bootstrap.InitiliazeDB)

	//Create admin user
	u, pass := assets.InsertAdminUser(api.mustDB())
	assert.NotZero(t, u)
	assert.NotZero(t, pass)

	//Insert Project
	pkey := sdk.RandomString(10)
	_ = assets.InsertTestProject(t, db, api.Cache, pkey, pkey, u)

	warn := sdk.WarningV2{
		Type:    warning.MissingProjectVariable,
		Key:     pkey,
		Created: time.Now(),
		Element: "sgu",
		MessageParams: map[string]string{
			"VarName":    "sgu",
			"ProjectKey": pkey,
			"EnvsName":   "Production, Staging",
			"AppsName":   "CDS, Venom",
		},
	}

	assert.NoError(t, warning.Insert(db, warn))

	v := map[string]string{
		"permProjectKey": pkey,
	}

	//Prepare request
	uri := api.Router.GetRoute("GET", api.getWarningsHandler, v)
	test.NotEmpty(t, uri)

	req := assets.NewAuthentifiedRequest(t, u, pass, "GET", uri, nil)

	//Do the request
	w := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)

	var ws []sdk.WarningV2
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &ws))

	assert.Equal(t, 1, len(ws))
	assert.Equal(t, "Variable sgu is used by Environments: \"Production, Staging\" and Applications: \"CDS, Venom\" but does not exist on project "+pkey, ws[0].Message)
}
