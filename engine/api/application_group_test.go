package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/sdk"
)

func Test_deleteGroupFromApplicationHandler(t *testing.T) {
	api, db, router := newTestAPI(t)

	//Create admin user
	u, pass := assets.InsertAdminUser(api.mustDB())

	gr := sdk.Group{
		Name:   "group-" + sdk.RandomString(5),
		Admins: []sdk.User{*u},
	}
	test.NoError(t, group.InsertGroup(db, &gr))

	if u.Groups == nil {
		u.Groups = []sdk.Group{gr}
	} else {
		u.Groups = append(u.Groups, gr)
	}

	//Insert Project
	pkey := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, api.Cache, pkey, pkey, u)

	//Insert Application
	app := &sdk.Application{
		Name: sdk.RandomString(10),
		ApplicationGroups: []sdk.GroupPermission{
			{Group: gr, Permission: 7},
		},
	}
	if err := application.Insert(db, api.Cache, proj, app, u); err != nil {
		t.Fatal(err)
	}
	test.NoError(t, group.InsertGroupInApplication(db, app.ID, gr.ID, 7))

	vars := map[string]string{
		"key": proj.Key,
		"permApplicationName": app.Name,
		"group":               gr.Name,
	}

	uri := router.GetRoute("DELETE", api.deleteGroupFromApplicationHandler, vars)

	req, err := http.NewRequest("DELETE", uri, nil)
	test.NoError(t, err)
	assets.AuthentifyRequest(t, req, u, pass)

	// Do the request
	w := httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)
	assert.Equal(t, 403, w.Code)
}
