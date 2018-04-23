package api

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/bootstrap"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/sdk"
)

func Test_getNavbarHandler(t *testing.T) {
	api, db, _ := newTestAPI(t, bootstrap.InitiliazeDB)

	u, pass := assets.InsertAdminUser(api.mustDB(context.Background()))

	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10), nil)
	app1 := sdk.Application{
		Name: "my-app-1",
	}
	app2 := sdk.Application{
		Name: "my-app-2",
	}
	test.NoError(t, application.Insert(db, api.Cache, proj, &app1, u))
	test.NoError(t, application.Insert(db, api.Cache, proj, &app2, u))

	//Prepare request
	uri := api.Router.GetRoute("GET", api.getNavbarHandler, nil)
	test.NotEmpty(t, uri)

	req := assets.NewAuthentifiedRequest(t, u, pass, "GET", uri, nil)

	//Do the request
	w := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)

	t.Logf("Body: %s", w.Body.String())
	data := []sdk.NavbarProjectData{}
	test.NoError(t, json.Unmarshal(w.Body.Bytes(), &data))

	var projFound, app1Found, app2Found bool
	for _, p := range data {
		if p.Key == proj.Key {
			projFound = true

			if p.ApplicationName == app1.Name {
				app1Found = true
				continue
			}

			if p.ApplicationName == app2.Name {
				app2Found = true
			}
		}
	}

	assert.True(t, projFound, "Project not found in handler response")
	assert.True(t, app1Found, "App1 not found in handler response")
	assert.True(t, app2Found, "App2 not found in handler response")
}
