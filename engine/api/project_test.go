package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/bootstrap"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/sdk"
	"github.com/stretchr/testify/assert"
)

func TestVariableInProject(t *testing.T) {
	api, db, _ := newTestAPI(t, bootstrap.InitiliazeDB)

	// 1. Create project
	project1 := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10), nil)

	// 2. Insert new variable
	var1 := &sdk.Variable{
		Name:  "var1",
		Value: "value1",
		Type:  "PASSWORD",
	}
	err := project.InsertVariable(api.mustDB(), project1, var1, &sdk.User{Username: "foo"})
	if err != nil {
		t.Fatalf("cannot insert var1 in project1: %s", err)
	}

	// 3. Test Update variable
	var1.Value = "value1Updated"
	err = project.UpdateVariable(api.mustDB(), project1, var1, &sdk.User{Username: "foo"})
	if err != nil {
		t.Fatalf("cannot update var1 in project1: %s", err)
	}

	// 4. Delete variable
	err = project.DeleteVariable(api.mustDB(), project1, var1, &sdk.User{Username: "foo"})
	if err != nil {
		t.Fatalf("cannot delete var1 from project: %s", err)
	}
	varTest, err := project.GetVariableInProject(api.mustDB(), project1.ID, var1.Name)
	if varTest.Value != "" {
		t.Fatalf("var1 should be deleted: %s", err)
	}

	// 5. Insert new var
	var2 := &sdk.Variable{
		Name:  "var2",
		Value: "value2",
		Type:  "STRING",
	}
	err = project.InsertVariable(api.mustDB(), project1, var2, &sdk.User{Username: "foo"})
	if err != nil {
		t.Fatalf("cannot insert var1 in project1: %s", err)
	}

}

func Test_getProjectsHandler(t *testing.T) {
	api, db, _ := newTestAPI(t, bootstrap.InitiliazeDB)
	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10), nil)
	repofullname := sdk.RandomString(10) + "/" + sdk.RandomString(10)
	app := &sdk.Application{
		Name:               "app",
		RepositoryFullname: repofullname,
	}
	u, pass := assets.InsertAdminUser(api.mustDB())
	test.NoError(t, application.Insert(db, api.Cache, proj, app, u))

	vars := map[string]string{}
	uri := api.Router.GetRoute("GET", api.getProjectsHandler, vars)
	uri += "?repo=" + url.QueryEscape(repofullname)
	req, err := http.NewRequest("GET", uri, nil)
	test.NoError(t, err)
	assets.AuthentifyRequest(t, req, u, pass)

	// Do the request
	w := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)

	projs := []sdk.Project{}
	test.NoError(t, json.Unmarshal(w.Body.Bytes(), &projs))
	assert.Len(t, projs, 1)
}
