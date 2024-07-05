package api

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/sdk"
	"github.com/stretchr/testify/require"
)

func Test_CrudProjectVariableSetItemText(t *testing.T) {
	api, db, _ := newTestAPI(t)

	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))
	user1, pass := assets.InsertLambdaUser(t, db)

	assets.InsertRBAcProject(t, db, sdk.ProjectRoleManage, proj.Key, *user1)
	assets.InsertRBAcProject(t, db, sdk.ProjectRoleRead, proj.Key, *user1)

	// INSERT Variable set
	vs := sdk.ProjectVariableSet{
		Name:       "vs-1",
		ProjectKey: proj.Key,
	}
	require.NoError(t, project.InsertVariableSet(context.TODO(), db, &vs))
	assets.InsertRBAcVariableSet(t, db, sdk.VariableSetRoleManage, proj.Key, "vs-1", *user1)

	it := sdk.ProjectVariableSetItem{
		Name:  "MyTextVar",
		Value: "MyTextValue",
		Type:  sdk.ProjectVariableTypeString,
	}

	vars := map[string]string{
		"projectKey":      proj.Key,
		"variableSetName": vs.Name,
	}
	uri := api.Router.GetRouteV2("POST", api.postProjectVariableSetItemHandler, vars)
	test.NotEmpty(t, uri)
	req := assets.NewAuthentifiedRequest(t, user1, pass, "POST", uri, &it)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w, req)
	require.Equal(t, 200, w.Code)

	// Update variable set item

	itUpdate := sdk.ProjectVariableSetItem{
		Name:  "MyTextVar",
		Value: "MyTextValueUpdated",
		Type:  sdk.ProjectVariableTypeString,
	}

	varsUpdate := map[string]string{
		"projectKey":      proj.Key,
		"variableSetName": vs.Name,
		"itemName":        itUpdate.Name,
	}
	uriUpdate := api.Router.GetRouteV2("PUT", api.putProjectVariableSetItemHandler, varsUpdate)
	test.NotEmpty(t, uriUpdate)
	reqUpdate := assets.NewAuthentifiedRequest(t, user1, pass, "PUT", uriUpdate, &itUpdate)
	reqUpdate.Header.Set("Content-Type", "application/json")

	wUpdate := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(wUpdate, reqUpdate)
	require.Equal(t, 200, wUpdate.Code)

	// Get variable set item
	varsGet1 := map[string]string{
		"projectKey":      proj.Key,
		"variableSetName": vs.Name,
		"itemName":        it.Name,
	}
	uriGet1 := api.Router.GetRouteV2("GET", api.getProjectVariableSetItemHandler, varsGet1)
	test.NotEmpty(t, uriGet1)
	reqGet1 := assets.NewAuthentifiedRequest(t, user1, pass, "GET", uriGet1, nil)
	wGet1 := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(wGet1, reqGet1)
	require.Equal(t, 200, wGet1.Code)

	var nGet sdk.ProjectVariableSetItem
	require.NoError(t, json.Unmarshal(wGet1.Body.Bytes(), &nGet))

	require.Equal(t, it.Name, nGet.Name)
	require.Equal(t, itUpdate.Value, nGet.Value)

	// Delete variable set
	varsDelete := map[string]string{
		"projectKey":      proj.Key,
		"variableSetName": vs.Name,
		"itemName":        it.Name,
	}
	uriDelete := api.Router.GetRouteV2("DELETE", api.deleteProjectVariableSetItemHandler, varsDelete)
	test.NotEmpty(t, uriDelete)
	reqDelete := assets.NewAuthentifiedRequest(t, user1, pass, "DELETE", uriDelete, nil)
	wDelete := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(wDelete, reqDelete)
	require.Equal(t, 204, wDelete.Code)

	vsDB, err := project.LoadVariableSetByName(context.TODO(), db, proj.Key, vs.Name)
	require.NoError(t, err)
	require.Equal(t, 0, len(vsDB.Items))
}

func Test_CrudProjectVariableSetItemSecret(t *testing.T) {
	api, db, _ := newTestAPI(t)

	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))
	user1, pass := assets.InsertLambdaUser(t, db)

	assets.InsertRBAcProject(t, db, sdk.ProjectRoleManage, proj.Key, *user1)
	assets.InsertRBAcProject(t, db, sdk.ProjectRoleRead, proj.Key, *user1)

	// INSERT Variable set
	vs := sdk.ProjectVariableSet{
		Name:       "vs-1",
		ProjectKey: proj.Key,
	}
	require.NoError(t, project.InsertVariableSet(context.TODO(), db, &vs))
	assets.InsertRBAcVariableSet(t, db, sdk.VariableSetRoleManage, proj.Key, "vs-1", *user1)

	it := sdk.ProjectVariableSetItem{
		Name:  "MyTextVar",
		Value: "MyTextValue",
		Type:  sdk.ProjectVariableTypeSecret,
	}

	vars := map[string]string{
		"projectKey":      proj.Key,
		"variableSetName": vs.Name,
	}
	uri := api.Router.GetRouteV2("POST", api.postProjectVariableSetItemHandler, vars)
	test.NotEmpty(t, uri)
	req := assets.NewAuthentifiedRequest(t, user1, pass, "POST", uri, &it)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w, req)
	require.Equal(t, 200, w.Code)

	// Update variable set item
	itUpdate := sdk.ProjectVariableSetItem{
		Name:  "MyTextVar",
		Value: "MyTextValueUpdated",
		Type:  sdk.ProjectVariableTypeSecret,
	}

	varsUpdate := map[string]string{
		"projectKey":      proj.Key,
		"variableSetName": vs.Name,
		"itemName":        itUpdate.Name,
	}
	uriUpdate := api.Router.GetRouteV2("PUT", api.putProjectVariableSetItemHandler, varsUpdate)
	test.NotEmpty(t, uriUpdate)
	reqUpdate := assets.NewAuthentifiedRequest(t, user1, pass, "PUT", uriUpdate, &itUpdate)
	reqUpdate.Header.Set("Content-Type", "application/json")

	wUpdate := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(wUpdate, reqUpdate)
	require.Equal(t, 200, wUpdate.Code)

	// Get variable set item
	varsGet1 := map[string]string{
		"projectKey":      proj.Key,
		"variableSetName": vs.Name,
		"itemName":        it.Name,
	}
	uriGet1 := api.Router.GetRouteV2("GET", api.getProjectVariableSetItemHandler, varsGet1)
	test.NotEmpty(t, uriGet1)
	reqGet1 := assets.NewAuthentifiedRequest(t, user1, pass, "GET", uriGet1, nil)
	wGet1 := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(wGet1, reqGet1)
	require.Equal(t, 200, wGet1.Code)

	var nGet sdk.ProjectVariableSetItem
	require.NoError(t, json.Unmarshal(wGet1.Body.Bytes(), &nGet))

	require.Equal(t, it.Name, nGet.Name)
	require.Equal(t, sdk.PasswordPlaceholder, nGet.Value)

	itDB, err := project.LoadVariableSetItem(context.TODO(), db, vs.ID, it.Name, gorpmapping.GetOptions.WithDecryption)
	require.NoError(t, err)
	require.Equal(t, itUpdate.Value, itDB.Value)

	// Delete variable set
	varsDelete := map[string]string{
		"projectKey":      proj.Key,
		"variableSetName": vs.Name,
		"itemName":        it.Name,
	}
	uriDelete := api.Router.GetRouteV2("DELETE", api.deleteProjectVariableSetItemHandler, varsDelete)
	test.NotEmpty(t, uriDelete)
	reqDelete := assets.NewAuthentifiedRequest(t, user1, pass, "DELETE", uriDelete, nil)
	wDelete := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(wDelete, reqDelete)
	require.Equal(t, 204, wDelete.Code)

	vsDB, err := project.LoadVariableSetByName(context.TODO(), db, proj.Key, vs.Name)
	require.NoError(t, err)
	require.Equal(t, 0, len(vsDB.Items))
}

func Test_CrudProjectVariableSetItemWithSameName(t *testing.T) {
	api, db, _ := newTestAPI(t)

	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))
	user1, pass := assets.InsertLambdaUser(t, db)

	assets.InsertRBAcProject(t, db, sdk.ProjectRoleManage, proj.Key, *user1)
	assets.InsertRBAcProject(t, db, sdk.ProjectRoleRead, proj.Key, *user1)

	// INSERT Variable set
	vs := sdk.ProjectVariableSet{
		Name:       "vs-1",
		ProjectKey: proj.Key,
	}
	require.NoError(t, project.InsertVariableSet(context.TODO(), db, &vs))
	assets.InsertRBAcVariableSet(t, db, sdk.VariableSetRoleManage, proj.Key, "vs-1", *user1)

	itSecret := sdk.ProjectVariableSetItem{
		Name:  "MyTextVar",
		Value: "MyTextValue",
		Type:  sdk.ProjectVariableTypeSecret,
	}

	vars := map[string]string{
		"projectKey":      proj.Key,
		"variableSetName": vs.Name,
	}
	uri := api.Router.GetRouteV2("POST", api.postProjectVariableSetItemHandler, vars)
	test.NotEmpty(t, uri)
	req := assets.NewAuthentifiedRequest(t, user1, pass, "POST", uri, &itSecret)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w, req)
	require.Equal(t, 200, w.Code)

	itText := sdk.ProjectVariableSetItem{
		Name:  "MyTextVar",
		Value: "MyTextValue",
		Type:  sdk.ProjectVariableTypeString,
	}

	varsText := map[string]string{
		"projectKey":      proj.Key,
		"variableSetName": vs.Name,
	}
	uriText := api.Router.GetRouteV2("POST", api.postProjectVariableSetItemHandler, varsText)
	test.NotEmpty(t, uriText)
	reqText := assets.NewAuthentifiedRequest(t, user1, pass, "POST", uriText, &itText)
	reqText.Header.Set("Content-Type", "application/json")

	wText := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(wText, reqText)
	require.Equal(t, 409, wText.Code)

}
