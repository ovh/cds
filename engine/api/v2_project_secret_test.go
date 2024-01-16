package api

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/ovh/cds/engine/api/project_secret"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
	"github.com/stretchr/testify/require"
)

func Test_crudProjectSecret(t *testing.T) {
	api, db, _ := newTestAPI(t)

	u, pass := assets.InsertAdminUser(t, db)
	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))

	ps := sdk.ProjectSecret{
		Name:  "mySecret",
		Value: "MySecret",
	}
	vars := map[string]string{
		"projectKey": proj.Key,
	}

	uriGet := api.Router.GetRouteV2("POST", api.postProjectSecretHandler, vars)
	test.NotEmpty(t, uriGet)
	req := assets.NewAuthentifiedRequest(t, u, pass, "POST", uriGet, ps)
	w := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w, req)
	require.Equal(t, 204, w.Code)

	// Then, update it
	varsPut := map[string]string{
		"projectKey": proj.Key,
		"name":       ps.Name,
	}
	psUpdated := sdk.ProjectSecret{
		Name:  "mySecret",
		Value: "MySecretUpdated",
	}
	uriPut := api.Router.GetRouteV2("PUT", api.putProjectSecretHandler, varsPut)
	test.NotEmpty(t, uriPut)
	reqPut := assets.NewAuthentifiedRequest(t, u, pass, "PUT", uriPut, psUpdated)
	wPut := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(wPut, reqPut)
	require.Equal(t, 204, wPut.Code)

	// List
	varsList := map[string]string{
		"projectKey": proj.Key,
	}
	uriList := api.Router.GetRouteV2("GET", api.getProjectSecretsHandler, varsList)
	test.NotEmpty(t, uriList)
	reqList := assets.NewAuthentifiedRequest(t, u, pass, "GET", uriList, nil)
	wList := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(wList, reqList)
	require.Equal(t, 200, wList.Code)

	var listSecrets []sdk.ProjectSecret
	require.NoError(t, json.Unmarshal(wList.Body.Bytes(), &listSecrets))
	require.Equal(t, 1, len(listSecrets))
	require.Equal(t, psUpdated.Name, listSecrets[0].Name)
	require.Equal(t, sdk.PasswordPlaceholder, listSecrets[0].Value)

	// Load from DB
	secretDB, err := project_secret.LoadByName(context.TODO(), db, proj.Key, psUpdated.Name, gorpmapper.GetOptions.WithDecryption)
	require.NoError(t, err)
	require.Equal(t, psUpdated.Value, secretDB.Value)

	secretsDB, err := project_secret.LoadByProjectKey(context.TODO(), db, proj.Key, gorpmapper.GetOptions.WithDecryption)
	require.NoError(t, err)
	require.Equal(t, psUpdated.Value, secretsDB[0].Value)

	// DELETE
	varsDelete := map[string]string{
		"projectKey": proj.Key,
		"name":       psUpdated.Name,
	}
	uriDel := api.Router.GetRouteV2("DELETE", api.deleteProjectSecretHandler, varsDelete)
	test.NotEmpty(t, uriDel)
	reqDel := assets.NewAuthentifiedRequest(t, u, pass, "DELETE", uriDel, nil)
	wDelete := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(wDelete, reqDel)
	require.Equal(t, 204, wDelete.Code)

	secrets, err := project_secret.LoadByProjectKey(context.TODO(), db, proj.Key)
	require.NoError(t, err)
	require.Equal(t, 0, len(secrets))
}
