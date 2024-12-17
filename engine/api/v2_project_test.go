package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/sdk"
)

func Test_getProjectsV2Handler(t *testing.T) {
	api, db, _ := newTestAPI(t)

	_ = assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))
	u, pass := assets.InsertAdminUser(t, db)

	vars := map[string]string{}
	uri := api.Router.GetRouteV2("GET", api.getProjectsV2Handler, vars)
	req, err := http.NewRequest("GET", uri, nil)
	test.NoError(t, err)
	assets.AuthentifyRequest(t, req, u, pass)

	// Do the request
	w := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w, req)
	require.Equal(t, 200, w.Code)

	projs := []sdk.Project{}
	test.NoError(t, json.Unmarshal(w.Body.Bytes(), &projs))
	require.Greater(t, len(projs), 0)
}

func Test_updateGetDeleteProjectV2Handler(t *testing.T) {
	api, db, _ := newTestAPI(t)

	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))
	u, pass := assets.InsertAdminUser(t, db)

	proj.Name = "updated_" + sdk.RandomString(10)

	// UPDATE
	vars := map[string]string{
		"projectKey": proj.Key,
	}

	bts, _ := json.Marshal(proj)
	reader := bytes.NewReader(bts)

	uri := api.Router.GetRouteV2("PUT", api.updateProjectV2Handler, vars)
	req, err := http.NewRequest("PUT", uri, reader)
	test.NoError(t, err)
	assets.AuthentifyRequest(t, req, u, pass)

	// Do the request
	w := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w, req)
	require.Equal(t, 200, w.Code)

	// GET
	uriGet := api.Router.GetRouteV2("GET", api.getProjectV2Handler, vars)
	reqGet, err := http.NewRequest("GET", uriGet, nil)
	test.NoError(t, err)
	assets.AuthentifyRequest(t, reqGet, u, pass)
	// Do the request
	wGET := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(wGET, reqGet)
	require.Equal(t, 200, wGET.Code)

	projGET := sdk.Project{}
	test.NoError(t, json.Unmarshal(wGET.Body.Bytes(), &projGET))
	require.Equal(t, proj.Name, projGET.Name)

	// DELETE
	uriDelete := api.Router.GetRouteV2("DELETE", api.deleteProjectV2Handler, vars)
	reqDel, err := http.NewRequest("DELETE", uriDelete, nil)
	test.NoError(t, err)
	assets.AuthentifyRequest(t, reqDel, u, pass)
	// Do the request
	wDel := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(wDel, reqDel)
	require.Equal(t, 204, wDel.Code)
}
