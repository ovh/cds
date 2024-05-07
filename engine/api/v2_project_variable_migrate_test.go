package api

import (
	"context"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
)

func TestPostMigrateApplicationVariables(t *testing.T) {
	api, db, _ := newTestAPI(t)
	ctx := context.TODO()

	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))
	user1, pass := assets.InsertLambdaUser(t, db)
	assets.InsertRBAcProject(t, db, sdk.ProjectRoleManage, proj.Key, *user1)
	assets.InsertRBAcProject(t, db, sdk.ProjectRoleRead, proj.Key, *user1)

	app := sdk.Application{Name: "myapp"}
	require.NoError(t, application.Insert(db, *proj, &app))
	v1 := sdk.ApplicationVariable{
		Name:          "v1",
		Value:         "value",
		Type:          "string",
		ApplicationID: app.ID,
	}
	require.NoError(t, application.InsertVariable(db, app.ID, &v1, user1))
	v2 := sdk.ApplicationVariable{
		Name:          "v2",
		Value:         "mysecretvalue",
		Type:          "password",
		ApplicationID: app.ID,
	}
	require.NoError(t, application.InsertVariable(db, app.ID, &v2, user1))

	copyRe := sdk.CopyApplicationVariableToVariableSet{
		ApplicationName: app.Name,
		VariableSetName: "newVarSet",
	}

	vars := map[string]string{
		"projectKey": proj.Key,
	}
	uri := api.Router.GetRouteV2("POST", api.postMigrateApplicationVariableToVariableSetHandler, vars)
	require.NotEmpty(t, uri)
	req := assets.NewAuthentifiedRequest(t, user1, pass, "POST", uri, &copyRe)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w, req)
	require.Equal(t, 200, w.Code)

	vs, err := project.LoadVariableSetByName(ctx, db, proj.Key, copyRe.VariableSetName)
	require.NoError(t, err)
	items, err := project.LoadVariableSetAllItem(ctx, db, vs.ID)
	require.NoError(t, err)

	var foundString, foundSecret bool
	for _, i := range items {
		if i.Type == sdk.ProjectVariableTypeString && i.Name == "v1" {
			require.Equal(t, v1.Value, i.Value)
			foundString = true
		}
		if i.Type == sdk.ProjectVariableTypeSecret && i.Name == "v2" {
			require.Equal(t, v2.Value, "**********")
			foundSecret = true
		}
	}
	require.True(t, foundSecret)
	require.True(t, foundString)

	secretItem, err := project.LoadVariableSetItem(ctx, db, vs.ID, "v2", gorpmapper.GetOptions.WithDecryption)
	require.NoError(t, err)
	require.Equal(t, "mysecretvalue", secretItem.Value)

}

func TestPostMigrateProjectVariableStringAndCreateVariableSet(t *testing.T) {
	api, db, _ := newTestAPI(t)
	ctx := context.TODO()

	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))
	user1, pass := assets.InsertLambdaUser(t, db)
	assets.InsertRBAcProject(t, db, sdk.ProjectRoleManage, proj.Key, *user1)
	assets.InsertRBAcProject(t, db, sdk.ProjectRoleRead, proj.Key, *user1)

	v1 := sdk.ProjectVariable{
		Name:      "v1",
		Value:     "value",
		Type:      "string",
		ProjectID: proj.ID,
	}
	require.NoError(t, project.InsertVariable(db, proj.ID, &v1, user1))

	copyRe := sdk.CopyProjectVariableToVariableSet{
		VariableName:    "v1",
		NewName:         "v11",
		VariableSetName: "newVarSet",
	}

	vars := map[string]string{
		"projectKey": proj.Key,
	}
	uri := api.Router.GetRouteV2("POST", api.postMigrateProjectVariableHandler, vars)
	require.NotEmpty(t, uri)
	req := assets.NewAuthentifiedRequest(t, user1, pass, "POST", uri+"?force=true", &copyRe)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w, req)
	require.Equal(t, 200, w.Code)

	vs, err := project.LoadVariableSetByName(ctx, db, proj.Key, copyRe.VariableSetName)
	require.NoError(t, err)
	items, err := project.LoadVariableSetAllItem(ctx, db, vs.ID)
	require.NoError(t, err)
	require.Len(t, items, 1)
	require.Equal(t, copyRe.NewName, items[0].Name)
	require.Equal(t, v1.Value, items[0].Value)
}

func TestPostMigrateProjectVariableStringVSDoNotExist(t *testing.T) {
	api, db, _ := newTestAPI(t)

	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))
	user1, pass := assets.InsertLambdaUser(t, db)
	assets.InsertRBAcProject(t, db, sdk.ProjectRoleManage, proj.Key, *user1)
	assets.InsertRBAcProject(t, db, sdk.ProjectRoleRead, proj.Key, *user1)

	v1 := sdk.ProjectVariable{
		Name:      "v1",
		Value:     "value",
		Type:      "string",
		ProjectID: proj.ID,
	}
	require.NoError(t, project.InsertVariable(db, proj.ID, &v1, user1))

	copyRe := sdk.CopyProjectVariableToVariableSet{
		VariableName:    "v1",
		NewName:         "v11",
		VariableSetName: "newVarSet",
	}

	vars := map[string]string{
		"projectKey": proj.Key,
	}
	uri := api.Router.GetRouteV2("POST", api.postMigrateProjectVariableHandler, vars)
	require.NotEmpty(t, uri)
	req := assets.NewAuthentifiedRequest(t, user1, pass, "POST", uri, &copyRe)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w, req)
	require.Equal(t, 404, w.Code)
}

func TestPostMigrateProjectVariableSecretAndCreateVariableSet(t *testing.T) {
	api, db, _ := newTestAPI(t)
	ctx := context.TODO()

	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))
	user1, pass := assets.InsertLambdaUser(t, db)
	assets.InsertRBAcProject(t, db, sdk.ProjectRoleManage, proj.Key, *user1)
	assets.InsertRBAcProject(t, db, sdk.ProjectRoleRead, proj.Key, *user1)

	v1 := sdk.ProjectVariable{
		Name:      "v1",
		Value:     "mysecretvalue",
		Type:      "password",
		ProjectID: proj.ID,
	}
	require.NoError(t, project.InsertVariable(db, proj.ID, &v1, user1))

	copyRe := sdk.CopyProjectVariableToVariableSet{
		VariableName:    "v1",
		VariableSetName: "newVarSet",
	}

	vars := map[string]string{
		"projectKey": proj.Key,
	}
	uri := api.Router.GetRouteV2("POST", api.postMigrateProjectVariableHandler, vars)
	require.NotEmpty(t, uri)
	req := assets.NewAuthentifiedRequest(t, user1, pass, "POST", uri+"?force=true", &copyRe)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w, req)
	require.Equal(t, 200, w.Code)

	vs, err := project.LoadVariableSetByName(ctx, db, proj.Key, copyRe.VariableSetName)
	require.NoError(t, err)
	itemNotClear, err := project.LoadVariableSetItem(ctx, db, vs.ID, "v1")
	require.NoError(t, err)
	require.Equal(t, "v1", itemNotClear.Name)
	require.Equal(t, "**********", itemNotClear.Value)

	itemClear, err := project.LoadVariableSetItem(ctx, db, vs.ID, "v1", gorpmapper.GetOptions.WithDecryption)
	require.NoError(t, err)
	require.Equal(t, "mysecretvalue", itemClear.Value)

}
