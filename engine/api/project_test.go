package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/authentication"
	"github.com/ovh/cds/engine/api/authentication/builtin"
	"github.com/ovh/cds/engine/api/bootstrap"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient"
)

func TestVariableInProject(t *testing.T) {
	api, db, _, end := newTestAPI(t, bootstrap.InitiliazeDB)
	defer end()

	// 1. Create project
	project1 := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10), nil)

	// 2. Insert new variable
	var1 := &sdk.Variable{
		Name:  "var1",
		Value: "value1",
		Type:  "PASSWORD",
	}
	err := project.InsertVariable(api.mustDB(), project1, var1, &sdk.AuthentifiedUser{Username: "foo"})
	if err != nil {
		t.Fatalf("cannot insert var1 in project1: %s", err)
	}

	// 3. Test Update variable
	var2 := var1
	var2.Value = "value1Updated"
	err = project.UpdateVariable(api.mustDB(), project1, var2, var1, &sdk.AuthentifiedUser{Username: "foo"})
	if err != nil {
		t.Fatalf("cannot update var1 in project1: %s", err)
	}

	// 4. Delete variable
	err = project.DeleteVariable(api.mustDB(), project1, var1, &sdk.AuthentifiedUser{Username: "foo"})
	if err != nil {
		t.Fatalf("cannot delete var1 from project: %s", err)
	}
	varTest, err := project.GetVariableInProject(api.mustDB(), project1.ID, var1.Name)
	if varTest.Value != "" {
		t.Fatalf("var1 should be deleted: %s", err)
	}

	// 5. Insert new var
	var3 := &sdk.Variable{
		Name:  "var2",
		Value: "value2",
		Type:  "STRING",
	}
	err = project.InsertVariable(api.mustDB(), project1, var3, &sdk.AuthentifiedUser{Username: "foo"})
	if err != nil {
		t.Fatalf("cannot insert var1 in project1: %s", err)
	}

}

func Test_getProjectsHandler(t *testing.T) {
	api, db, _, end := newTestAPI(t, bootstrap.InitiliazeDB)
	defer end()
	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10), nil)
	repofullname := sdk.RandomString(10) + "/" + sdk.RandomString(10)
	app := &sdk.Application{
		Name:               "app",
		RepositoryFullname: repofullname,
	}
	u, pass := assets.InsertAdminUser(api.mustDB())
	test.NoError(t, application.Insert(db, api.Cache, proj, app))

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

func Test_addProjectHandler(t *testing.T) {
	api, db, _, end := newTestAPI(t, bootstrap.InitiliazeDB)
	defer end()
	u, pass := assets.InsertLambdaUser(db)

	proj := sdk.Project{
		Key:  strings.ToUpper(sdk.RandomString(15)),
		Name: sdk.RandomString(15),
	}

	jsonBody, _ := json.Marshal(proj)
	body := bytes.NewBuffer(jsonBody)

	uri := api.Router.GetRoute("POST", api.addProjectHandler, nil)
	req, err := http.NewRequest("POST", uri, body)
	test.NoError(t, err)
	assets.AuthentifyRequest(t, req, u, pass)

	// Do the request
	w := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w, req)
	assert.Equal(t, 201, w.Code)

	projCreated := sdk.Project{}
	test.NoError(t, json.Unmarshal(w.Body.Bytes(), &projCreated))
	assert.Equal(t, proj.Key, projCreated.Key)

	gr, err := group.LoadByName(context.TODO(), db, proj.Name)
	assert.NotNil(t, gr)
	assert.NoError(t, err)
}

func Test_addProjectHandlerWithGroup(t *testing.T) {
	api, db, _, end := newTestAPI(t, bootstrap.InitiliazeDB)
	defer end()
	u, pass := assets.InsertAdminUser(db)
	g := sdk.Group{Name: sdk.RandomString(10)}
	test.NoError(t, group.InsertGroup(db, &g))

	proj := sdk.Project{
		Key:  strings.ToUpper(sdk.RandomString(10)),
		Name: sdk.RandomString(10),
		ProjectGroups: []sdk.GroupPermission{
			{Group: sdk.Group{Name: g.Name}},
		},
	}

	jsonBody, _ := json.Marshal(proj)
	body := bytes.NewBuffer(jsonBody)

	uri := api.Router.GetRoute("POST", api.addProjectHandler, nil)
	req, err := http.NewRequest("POST", uri, body)
	test.NoError(t, err)
	assets.AuthentifyRequest(t, req, u, pass)

	// Do the request
	w := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w, req)
	assert.Equal(t, 201, w.Code)

	projCreated := sdk.Project{}
	test.NoError(t, json.Unmarshal(w.Body.Bytes(), &projCreated))
	assert.Equal(t, proj.Key, projCreated.Key)

	gr, err := group.LoadByName(context.TODO(), db, proj.Name)
	assert.Nil(t, gr)
	assert.Error(t, err)
}

func Test_getProjectsHandler_WithWPermissionShouldReturnNoProjects(t *testing.T) {
	api, db, _, end := newTestAPI(t, bootstrap.InitiliazeDB)
	defer end()
	assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10), nil)

	u, pass := assets.InsertLambdaUser(api.mustDB())

	vars := map[string]string{}
	uri := api.Router.GetRoute("GET", api.getProjectsHandler, vars)
	uri += "?permission=W"
	req, err := http.NewRequest("GET", uri, nil)
	test.NoError(t, err)
	assets.AuthentifyRequest(t, req, u, pass)

	// Do the request
	w := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)

	projs := []sdk.Project{}
	test.NoError(t, json.Unmarshal(w.Body.Bytes(), &projs))
	assert.Len(t, projs, 0, "too much project")
}

func Test_getProjectsHandler_WithWPermissionShouldReturnOneProject(t *testing.T) {
	api, db, _, end := newTestAPI(t, bootstrap.InitiliazeDB)
	defer end()
	u, pass := assets.InsertLambdaUser(api.mustDB())
	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10), u)
	test.NoError(t, group.InsertUserInGroup(db, proj.ProjectGroups[0].Group.ID, u.OldUserStruct.ID, true))

	vars := map[string]string{}
	uri := api.Router.GetRoute("GET", api.getProjectsHandler, vars)
	uri += "?permission=W"
	req, err := http.NewRequest("GET", uri, nil)
	test.NoError(t, err)
	assets.AuthentifyRequest(t, req, u, pass)

	// Do the request
	w := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)

	projs := []sdk.Project{}
	test.NoError(t, json.Unmarshal(w.Body.Bytes(), &projs))
	assert.Len(t, projs, 1, "should have one project")
}

func Test_getprojectsHandler_AsProvider(t *testing.T) {
	api, tsURL, tsClose := newTestServer(t)
	defer tsClose()

	admin, _ := assets.InsertAdminUser(api.mustDB())
	localConsumer, err := authentication.LoadConsumerByTypeAndUserID(context.TODO(), api.mustDB(), sdk.ConsumerLocal, admin.ID, authentication.LoadConsumerOptions.WithAuthentifiedUser)
	require.NoError(t, err)

	_, jws, err := builtin.NewConsumer(api.mustDB(), sdk.RandomString(10), sdk.RandomString(10), localConsumer, admin.GetGroupIDs(), Scope(sdk.AuthConsumerScopeProject))
	require.NoError(t, err)

	u, _ := assets.InsertLambdaUser(api.mustDB())
	pkey := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, api.mustDB(), api.Cache, pkey, pkey, u)
	test.NoError(t, group.InsertUserInGroup(api.mustDB(), proj.ProjectGroups[0].Group.ID, u.OldUserStruct.ID, true))

	sdkclient := cdsclient.NewProviderClient(cdsclient.ProviderConfig{
		Host:  tsURL,
		Token: jws,
	})

	projs, err := sdkclient.ProjectsList()
	test.NoError(t, err)
	assert.True(t, len(projs) > 0)
}

func Test_getprojectsHandler_AsProviderWithRequestedUsername(t *testing.T) {
	api, tsURL, tsClose := newTestServer(t)
	defer tsClose()

	admin, _ := assets.InsertAdminUser(api.mustDB())
	localConsumer, err := authentication.LoadConsumerByTypeAndUserID(context.TODO(), api.mustDB(), sdk.ConsumerLocal, admin.ID, authentication.LoadConsumerOptions.WithAuthentifiedUser)
	require.NoError(t, err)

	_, jws, err := builtin.NewConsumer(api.mustDB(), sdk.RandomString(10), sdk.RandomString(10), localConsumer, admin.GetGroupIDs(), Scope(sdk.AuthConsumerScopeProject))

	u, _ := assets.InsertLambdaUser(api.mustDB())

	pkey := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, api.mustDB(), api.Cache, pkey, pkey, u)
	test.NoError(t, group.InsertUserInGroup(api.mustDB(), proj.ProjectGroups[0].Group.ID, u.OldUserStruct.ID, true))

	app := &sdk.Application{
		Name: sdk.RandomString(10),
	}
	test.NoError(t, application.Insert(api.mustDB(), api.Cache, proj, app))

	sdkclient := cdsclient.NewProviderClient(cdsclient.ProviderConfig{
		Host:  tsURL,
		Token: jws,
	})

	projs, err := sdkclient.ProjectsList(cdsclient.FilterByUser(u.Username))
	test.NoError(t, err)
	assert.Len(t, projs, 1)

	apps, err := sdkclient.ApplicationsList(pkey, cdsclient.FilterByUser(u.Username), cdsclient.WithUsage())
	test.NoError(t, err)
	assert.Len(t, apps, 1)
}

func Test_putProjectLabelsHandler(t *testing.T) {
	api, db, _, end := newTestAPI(t, bootstrap.InitiliazeDB)
	defer end()
	u, pass := assets.InsertAdminUser(db)

	pkey := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, api.mustDB(), api.Cache, pkey, pkey, u)
	test.NoError(t, group.InsertUserInGroup(api.mustDB(), proj.ProjectGroups[0].Group.ID, u.OldUserStruct.ID, true))

	lbl1 := sdk.Label{
		Name:      sdk.RandomString(5),
		ProjectID: proj.ID,
	}
	test.NoError(t, project.InsertLabel(db, &lbl1))
	lbl2 := sdk.Label{
		Name:      sdk.RandomString(5),
		ProjectID: proj.ID,
	}
	test.NoError(t, project.InsertLabel(db, &lbl2))

	bodyLabels := []sdk.Label{
		{ID: lbl1.ID, Name: "this is a test", Color: lbl1.Color},
		{Name: "anotherone"},
		{Name: "anotheronebis", Color: "#FF0000"},
	}
	jsonBody, _ := json.Marshal(bodyLabels)
	body := bytes.NewBuffer(jsonBody)
	vars := map[string]string{
		"permProjectKey": proj.Key,
	}
	uri := api.Router.GetRoute("PUT", api.putProjectLabelsHandler, vars)
	req, err := http.NewRequest("PUT", uri, body)
	test.NoError(t, err)
	assets.AuthentifyRequest(t, req, u, pass)

	// Do the request
	w := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)

	projReturned := sdk.Project{}
	test.NoError(t, json.Unmarshal(w.Body.Bytes(), &projReturned))
	assert.Equal(t, proj.Key, projReturned.Key)
	assert.NotNil(t, projReturned.Labels)
	assert.Equal(t, 3, len(projReturned.Labels))
	assert.Equal(t, "anotherone", projReturned.Labels[0].Name)
	assert.NotZero(t, projReturned.Labels[0].Color)
	assert.Equal(t, "anotheronebis", projReturned.Labels[1].Name)
	assert.NotZero(t, projReturned.Labels[1].Color)
	assert.Equal(t, "#FF0000", projReturned.Labels[1].Color)
	assert.Equal(t, "this is a test", projReturned.Labels[2].Name)
	assert.NotZero(t, projReturned.Labels[2].Color)
}
