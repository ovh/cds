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
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient"
)

func TestVariableInProject(t *testing.T) {
	api, db, _ := newTestAPI(t)

	// 1. Create project
	project1 := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))

	// 2. Insert new variable
	var1 := &sdk.ProjectVariable{
		Name:  "var1",
		Value: "value1",
		Type:  "PASSWORD",
	}
	require.NoError(t, project.InsertVariable(db, project1.ID, var1, &sdk.AuthentifiedUser{Username: "foo"}))

	// 3. Test Update variable
	var2 := var1
	var2.Value = "value1Updated"
	require.NoError(t, project.UpdateVariable(db, project1.ID, var2, var1, &sdk.AuthentifiedUser{Username: "foo"}))

	// 4. Delete variable
	require.NoError(t, project.DeleteVariable(api.mustDB(), project1.ID, var1, &sdk.AuthentifiedUser{Username: "foo"}))
	_, err := project.LoadVariable(api.mustDB(), project1.ID, var1.Name)
	require.Error(t, err)

	// 5. Insert new var
	var3 := &sdk.ProjectVariable{
		Name:  "var2",
		Value: "value2",
		Type:  "STRING",
	}
	require.NoError(t, project.InsertVariable(db, project1.ID, var3, &sdk.AuthentifiedUser{Username: "foo"}))
}

func Test_getProjectsHandler(t *testing.T) {
	api, db, _ := newTestAPI(t)

	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))
	repofullname := sdk.RandomString(10) + "/" + sdk.RandomString(10)
	app := &sdk.Application{
		Name:               "app",
		RepositoryFullname: repofullname,
	}
	u, pass := assets.InsertAdminUser(t, db)
	test.NoError(t, application.Insert(db, *proj, app))

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
	api, db, _ := newTestAPI(t)

	u, pass := assets.InsertLambdaUser(t, db)

	proj := sdk.Project{
		Key:  strings.ToUpper(sdk.RandomString(15)),
		Name: sdk.RandomString(15),
	}

	jsonBody, _ := json.Marshal(proj)
	body := bytes.NewBuffer(jsonBody)

	uri := api.Router.GetRoute("POST", api.postProjectHandler, nil)
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
	assert.Equal(t, true, projCreated.Permissions.Writable)

	gr, err := group.LoadByName(context.TODO(), db, proj.Name)
	assert.NotNil(t, gr)
	assert.NoError(t, err)
}

func Test_addProjectHandlerWithGroup(t *testing.T) {
	api, db, _ := newTestAPI(t)

	u, pass := assets.InsertAdminUser(t, db)
	g := sdk.Group{Name: sdk.RandomString(10)}
	require.NoError(t, group.Insert(context.TODO(), db, &g))

	proj := sdk.Project{
		Key:  strings.ToUpper(sdk.RandomString(10)),
		Name: sdk.RandomString(10),
		ProjectGroups: []sdk.GroupPermission{
			{Group: sdk.Group{Name: g.Name}},
		},
	}

	jsonBody, _ := json.Marshal(proj)
	body := bytes.NewBuffer(jsonBody)

	uri := api.Router.GetRoute("POST", api.postProjectHandler, nil)
	req, err := http.NewRequest("POST", uri, body)
	require.NoError(t, err)
	assets.AuthentifyRequest(t, req, u, pass)

	// Do the request
	w := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w, req)
	assert.Equal(t, 201, w.Code)

	var projCreated sdk.Project
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &projCreated))
	assert.Equal(t, proj.Key, projCreated.Key)

	_, err = group.LoadByName(context.TODO(), db, proj.Name)
	assert.True(t, sdk.ErrorIs(err, sdk.ErrNotFound), "no group should have been created")
}

func Test_getProjectsHandler_WithWPermissionShouldReturnNoProjects(t *testing.T) {
	api, db, _ := newTestAPI(t)

	assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))

	u, pass := assets.InsertLambdaUser(t, db)

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

func Test_getProjectHandler_CheckPermission(t *testing.T) {
	api, db, _ := newTestAPI(t)

	u, pass := assets.InsertLambdaUser(t, db)
	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))
	require.NoError(t, group.InsertLinkGroupUser(context.TODO(), db, &group.LinkGroupUser{
		GroupID:            proj.ProjectGroups[0].Group.ID,
		AuthentifiedUserID: u.ID,
		Admin:              true,
	}))

	vars := map[string]string{
		"permProjectKey": proj.Key,
	}
	uri := api.Router.GetRoute("GET", api.getProjectHandler, vars)
	req, err := http.NewRequest("GET", uri, nil)
	test.NoError(t, err)
	assets.AuthentifyRequest(t, req, u, pass)

	// Do the request
	w := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)

	projGet := sdk.Project{}
	test.NoError(t, json.Unmarshal(w.Body.Bytes(), &projGet))
	assert.Equal(t, true, projGet.Permissions.Readable, "readable should be true")
	assert.Equal(t, true, projGet.Permissions.Writable, "writable should be true")

	userAdmin, passAdmin := assets.InsertAdminUser(t, db)
	uri = api.Router.GetRoute("GET", api.getProjectHandler, vars)
	req, err = http.NewRequest("GET", uri, nil)
	test.NoError(t, err)
	assets.AuthentifyRequest(t, req, userAdmin, passAdmin)

	// Do the request
	w = httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)

	projGet = sdk.Project{}
	test.NoError(t, json.Unmarshal(w.Body.Bytes(), &projGet))
	assert.Equal(t, true, projGet.Permissions.Readable, "readable should be true")
	assert.Equal(t, true, projGet.Permissions.Writable, "writable should be true")

	userMaintainer, passMaintainer := assets.InsertMaintainerUser(t, db)
	uri = api.Router.GetRoute("GET", api.getProjectHandler, vars)
	req, err = http.NewRequest("GET", uri, nil)
	test.NoError(t, err)
	assets.AuthentifyRequest(t, req, userMaintainer, passMaintainer)

	// Do the request
	w = httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)

	projGet = sdk.Project{}
	test.NoError(t, json.Unmarshal(w.Body.Bytes(), &projGet))
	assert.Equal(t, true, projGet.Permissions.Readable, "readable should be true")
	assert.Equal(t, false, projGet.Permissions.Writable, "writable should be false")
}

func Test_getProjectsHandler_WithWPermissionShouldReturnOneProject(t *testing.T) {
	api, db, _ := newTestAPI(t)

	u, pass := assets.InsertLambdaUser(t, db)
	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))
	require.NoError(t, group.InsertLinkGroupUser(context.TODO(), db, &group.LinkGroupUser{
		GroupID:            proj.ProjectGroups[0].Group.ID,
		AuthentifiedUserID: u.ID,
		Admin:              true,
	}))

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

func Test_getProjectsHandler_AsProvider(t *testing.T) {
	api, db, tsURL := newTestServer(t)

	admin, _ := assets.InsertAdminUser(t, db)
	localConsumer, err := authentication.LoadConsumerByTypeAndUserID(context.TODO(), api.mustDB(), sdk.ConsumerLocal, admin.ID, authentication.LoadConsumerOptions.WithAuthentifiedUser)
	require.NoError(t, err)

	_, jws, err := builtin.NewConsumer(context.TODO(), db, sdk.RandomString(10), sdk.RandomString(10), 0, localConsumer, admin.GetGroupIDs(),
		sdk.NewAuthConsumerScopeDetails(sdk.AuthConsumerScopeProject))
	require.NoError(t, err)

	u, _ := assets.InsertLambdaUser(t, db)
	pkey := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, api.Cache, pkey, pkey)
	require.NoError(t, group.InsertLinkGroupUser(context.TODO(), db, &group.LinkGroupUser{
		GroupID:            proj.ProjectGroups[0].Group.ID,
		AuthentifiedUserID: u.ID,
		Admin:              true,
	}))

	sdkclient := cdsclient.NewProviderClient(cdsclient.ProviderConfig{
		Host:  tsURL,
		Token: jws,
	})

	projs, err := sdkclient.ProjectsList()
	test.NoError(t, err)
	assert.True(t, len(projs) > 0)
}

func Test_getprojectsHandler_AsProviderWithRequestedUsername(t *testing.T) {
	api, db, tsURL := newTestServer(t)

	admin, _ := assets.InsertAdminUser(t, db)
	localConsumer, err := authentication.LoadConsumerByTypeAndUserID(context.TODO(), api.mustDB(), sdk.ConsumerLocal, admin.ID, authentication.LoadConsumerOptions.WithAuthentifiedUser)
	require.NoError(t, err)

	_, jws, err := builtin.NewConsumer(context.TODO(), db, sdk.RandomString(10), sdk.RandomString(10), 0, localConsumer, admin.GetGroupIDs(),
		sdk.NewAuthConsumerScopeDetails(sdk.AuthConsumerScopeProject))
	require.NoError(t, err)

	u, _ := assets.InsertLambdaUser(t, db)

	pkey := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, api.Cache, pkey, pkey)
	require.NoError(t, group.InsertLinkGroupUser(context.TODO(), db, &group.LinkGroupUser{
		GroupID:            proj.ProjectGroups[0].Group.ID,
		AuthentifiedUserID: u.ID,
		Admin:              true,
	}))

	app := &sdk.Application{
		Name: sdk.RandomString(10),
	}
	require.NoError(t, application.Insert(db, *proj, app))

	// Call with an admin
	sdkclientAdmin := cdsclient.NewProviderClient(cdsclient.ProviderConfig{
		Host:  tsURL,
		Token: jws,
	})

	projs, err := sdkclientAdmin.ProjectsList()
	require.NoError(t, err)
	assert.True(t, len(projs) >= 1)

	apps, err := sdkclientAdmin.ApplicationsList(pkey, cdsclient.FilterByUser(u.Username), cdsclient.WithUsage(), cdsclient.FilterByWritablePermission())
	require.NoError(t, err)
	assert.True(t, len(apps) >= 1)

	// Call like a provider
	sdkclientProvider := cdsclient.NewProviderClient(cdsclient.ProviderConfig{
		Host:  tsURL,
		Token: jws,
	})

	projs, err = sdkclientProvider.ProjectsList(cdsclient.FilterByUser(u.Username), cdsclient.FilterByWritablePermission())
	require.NoError(t, err)
	assert.Len(t, projs, 1)

	apps, err = sdkclientProvider.ApplicationsList(pkey, cdsclient.FilterByUser(u.Username), cdsclient.WithUsage(), cdsclient.FilterByWritablePermission())
	require.NoError(t, err)
	assert.Len(t, apps, 1)
}

func Test_putProjectLabelsHandler(t *testing.T) {
	api, db, _ := newTestAPI(t)

	u, pass := assets.InsertAdminUser(t, db)

	pkey := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, api.Cache, pkey, pkey)
	require.NoError(t, group.InsertLinkGroupUser(context.TODO(), db, &group.LinkGroupUser{
		GroupID:            proj.ProjectGroups[0].Group.ID,
		AuthentifiedUserID: u.ID,
		Admin:              true,
	}))

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

func Test_getProjectsHandler_FilterByRepo(t *testing.T) {
	api, db, tsURL := newTestServer(t)

	admin, _ := assets.InsertAdminUser(t, db)
	localConsumer, err := authentication.LoadConsumerByTypeAndUserID(context.TODO(), api.mustDB(), sdk.ConsumerLocal, admin.ID, authentication.LoadConsumerOptions.WithAuthentifiedUser)
	require.NoError(t, err)

	_, jws, err := builtin.NewConsumer(context.TODO(), db, sdk.RandomString(10), sdk.RandomString(10), 0, localConsumer, admin.GetGroupIDs(),
		sdk.NewAuthConsumerScopeDetails(sdk.AuthConsumerScopeProject))
	require.NoError(t, err)

	u, _ := assets.InsertLambdaUser(t, db)

	pkey := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, api.Cache, pkey, pkey)
	require.NoError(t, group.InsertLinkGroupUser(context.TODO(), db, &group.LinkGroupUser{
		GroupID:            proj.ProjectGroups[0].Group.ID,
		AuthentifiedUserID: u.ID,
		Admin:              true,
	}))

	repofullName := sdk.RandomString(10)

	app := &sdk.Application{
		Name:               sdk.RandomString(10),
		RepositoryFullname: "ovh/" + repofullName,
	}
	require.NoError(t, application.Insert(db, *proj, app))

	pip := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip1",
	}
	test.NoError(t, pipeline.InsertPipeline(api.mustDB(), &pip))

	wf := sdk.Workflow{
		Name:       "workflow1",
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		WorkflowData: sdk.WorkflowData{
			Node: sdk.Node{
				Name: "root",
				Context: &sdk.NodeContext{
					PipelineID:    pip.ID,
					ApplicationID: app.ID,
				},
			},
		},
	}
	test.NoError(t, workflow.Insert(context.TODO(), db, api.Cache, *proj, &wf))

	// Call with an admin
	sdkclientAdmin := cdsclient.New(cdsclient.Config{
		Host:                              tsURL,
		BuitinConsumerAuthenticationToken: jws,
	})

	projs, err := sdkclientAdmin.ProjectList(true, true, cdsclient.Filter{Name: "repo", Value: "ovh/" + repofullName})
	require.NoError(t, err)
	require.True(t, len(projs) == 1)
	require.True(t, len(projs[0].Workflows) == 1)
	require.Equal(t, wf.Name, projs[0].Workflows[0].Name)
	require.Equal(t, app.ID, projs[0].Workflows[0].WorkflowData.Node.Context.ApplicationID)
	require.Equal(t, pip.ID, projs[0].Workflows[0].WorkflowData.Node.Context.PipelineID)

}
