package api

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/sdk"
	"github.com/stretchr/testify/require"
)

func Test_CrudProjectVariableSet(t *testing.T) {
	api, db, _ := newTestAPI(t)

	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))
	user1, pass := assets.InsertLambdaUser(t, db)

	assets.InsertRBAcProject(t, db, sdk.ProjectRoleManage, proj.Key, *user1)
	assets.InsertRBAcProject(t, db, sdk.ProjectRoleRead, proj.Key, *user1)
	assets.InsertRBAcVariableSet(t, db, sdk.VariableSetRoleManage, proj.Key, "vs-1", *user1)

	// INSERT Variable set
	vs := sdk.ProjectVariableSet{
		Name: "vs-1",
	}

	vars := map[string]string{
		"projectKey": proj.Key,
	}
	uri := api.Router.GetRouteV2("POST", api.postProjectVariableSetHandler, vars)
	test.NotEmpty(t, uri)
	req := assets.NewAuthentifiedRequest(t, user1, pass, "POST", uri, vs)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w, req)
	require.Equal(t, 200, w.Code)

	// List variable set
	varsList := map[string]string{
		"projectKey": proj.Key,
	}
	uriList := api.Router.GetRouteV2("GET", api.getProjectVariableSetsHandler, varsList)
	test.NotEmpty(t, uriList)
	reqList := assets.NewAuthentifiedRequest(t, user1, pass, "GET", uriList, nil)
	wList := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(wList, reqList)
	require.Equal(t, 200, wList.Code)

	var vss []sdk.ProjectVariableSet
	require.NoError(t, json.Unmarshal(wList.Body.Bytes(), &vss))

	require.Equal(t, 1, len(vss))
	require.Equal(t, vss[0].Name, vs.Name)

	// Get variable set
	varsGet1 := map[string]string{
		"projectKey":      proj.Key,
		"variableSetName": vs.Name,
	}
	uriGet1 := api.Router.GetRouteV2("GET", api.getProjectVariableSetHandler, varsGet1)
	test.NotEmpty(t, uriGet1)
	reqGet1 := assets.NewAuthentifiedRequest(t, user1, pass, "GET", uriGet1, nil)
	wGet1 := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(wGet1, reqGet1)
	require.Equal(t, 200, wGet1.Code)

	var nGet sdk.ProjectVariableSet
	require.NoError(t, json.Unmarshal(wGet1.Body.Bytes(), &nGet))

	require.Equal(t, vs.Name, nGet.Name)

	// Delete variable set
	varsDelete := map[string]string{
		"projectKey":      proj.Key,
		"variableSetName": vs.Name,
	}
	uriDelete := api.Router.GetRouteV2("DELETE", api.deleteProjectVariableSetHandler, varsDelete)
	test.NotEmpty(t, uriDelete)
	reqDelete := assets.NewAuthentifiedRequest(t, user1, pass, "DELETE", uriDelete, nil)
	wDelete := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(wDelete, reqDelete)
	require.Equal(t, 204, wDelete.Code)

	vssDB, err := project.LoadVariableSetsByProject(context.TODO(), db, proj.Key)
	require.NoError(t, err)
	require.Equal(t, 0, len(vssDB))
}

func TestLoadVariableSetWithItems(t *testing.T) {
	api, db, _ := newTestAPI(t)

	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))
	user1, pass := assets.InsertLambdaUser(t, db)

	assets.InsertRBAcProject(t, db, sdk.ProjectRoleManage, proj.Key, *user1)
	assets.InsertRBAcProject(t, db, sdk.ProjectRoleRead, proj.Key, *user1)

	// Create variable SET
	vs := sdk.ProjectVariableSet{
		ProjectKey: proj.Key,
		Name:       sdk.RandomString(10),
	}
	require.NoError(t, project.InsertVariableSet(context.TODO(), db, &vs))

	assets.InsertRBAcVariableSet(t, db, sdk.VariableSetRoleManage, proj.Key, vs.Name, *user1)

	// ADD ITEM TEXT
	itt := sdk.ProjectVariableSetItem{
		ProjectVariableSetID: vs.ID,
		Name:                 sdk.RandomString(10),
		Type:                 sdk.ProjectVariableTypeString,
		Value:                "myValue",
	}
	require.NoError(t, project.InsertVariableSetItemText(context.TODO(), db, &itt))

	// ADD ITEM Secret
	its := sdk.ProjectVariableSetItem{
		ProjectVariableSetID: vs.ID,
		Name:                 sdk.RandomString(10),
		Type:                 sdk.ProjectVariableTypeSecret,
		Value:                "mySecretValue",
	}
	require.NoError(t, project.InsertVariableSetItemSecret(context.TODO(), db, &its))

	// Get variable set
	varsGet1 := map[string]string{
		"projectKey":      proj.Key,
		"variableSetName": vs.Name,
	}
	uriGet1 := api.Router.GetRouteV2("GET", api.getProjectVariableSetHandler, varsGet1)
	test.NotEmpty(t, uriGet1)
	reqGet1 := assets.NewAuthentifiedRequest(t, user1, pass, "GET", uriGet1, nil)
	wGet1 := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(wGet1, reqGet1)
	require.Equal(t, 200, wGet1.Code)

	var nGet sdk.ProjectVariableSet
	require.NoError(t, json.Unmarshal(wGet1.Body.Bytes(), &nGet))

	require.Equal(t, 2, len(nGet.Items))
	require.Equal(t, itt.Value, nGet.Items[0].Value)
	require.Equal(t, sdk.PasswordPlaceholder, nGet.Items[1].Value)

}
