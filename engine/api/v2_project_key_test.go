package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ovh/cds/engine/api/keys"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/sdk"
	"github.com/stretchr/testify/require"
)

func Test_getKeysInProjectV2Handler(t *testing.T) {
	api, db, router := newTestAPI(t)

	//Create admin user
	u, pass := assets.InsertAdminUser(t, db)

	//Insert Project
	pkey := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, api.Cache, pkey, pkey)

	k := &sdk.ProjectKey{
		Name:      "mykey",
		Type:      "pgp",
		ProjectID: proj.ID,
	}

	kpgp, err := keys.GeneratePGPKeyPair(k.Name, "", "test@cds")
	if err != nil {
		t.Fatal(err)
	}
	k.KeyID = kpgp.KeyID
	k.Public = kpgp.Public
	k.Private = kpgp.Private
	k.Type = kpgp.Type

	if err := project.InsertKey(db, k); err != nil {
		t.Fatal(err)
	}

	vars := map[string]string{
		"projectKey": proj.Key,
		"name":       k.Name,
	}

	uri := router.GetRouteV2("GET", api.getKeysInProjectV2Handler, vars)
	req, err := http.NewRequest("GET", uri, nil)
	test.NoError(t, err)
	assets.AuthentifyRequest(t, req, u, pass)

	// Do the request
	w := httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)
	require.Equal(t, 200, w.Code)

	var keys []sdk.ProjectKey
	test.NoError(t, json.Unmarshal(w.Body.Bytes(), &keys))
	require.Equal(t, len(keys), 1)

	uri = router.GetRouteV2("POST", api.postDisableKeyInProjectV2Handler, vars)
	req, err = http.NewRequest("POST", uri, nil)
	test.NoError(t, err)
	assets.AuthentifyRequest(t, req, u, pass)
	w = httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)
	require.Equal(t, 200, w.Code)

	uri = router.GetRouteV2("POST", api.postEnableKeyInProjectV2Handler, vars)
	req, err = http.NewRequest("POST", uri, nil)
	test.NoError(t, err)
	assets.AuthentifyRequest(t, req, u, pass)
	w = httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)
	require.Equal(t, 200, w.Code)
}

func Test_deleteKeyInProjectV2Handler(t *testing.T) {
	api, db, router := newTestAPI(t)

	//Create admin user
	u, pass := assets.InsertAdminUser(t, db)

	//Insert Project
	pkey := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, api.Cache, pkey, pkey)

	k := &sdk.ProjectKey{
		Name:      "mykey",
		Type:      "pgp",
		Public:    "pub",
		Private:   "priv",
		ProjectID: proj.ID,
	}

	if err := project.InsertKey(db, k); err != nil {
		t.Fatal(err)
	}

	vars := map[string]string{
		"projectKey": proj.Key,
		"name":       k.Name,
	}

	uri := router.GetRouteV2("DELETE", api.deleteKeyInProjectV2Handler, vars)
	req, err := http.NewRequest("DELETE", uri, nil)
	test.NoError(t, err)
	assets.AuthentifyRequest(t, req, u, pass)

	// Do the request
	w := httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)
	require.Equal(t, 200, w.Code)

	var keys []sdk.ProjectKey
	test.NoError(t, json.Unmarshal(w.Body.Bytes(), &keys))
	require.Equal(t, len(keys), 0)
}

func Test_addKeyInProjectV2Handler(t *testing.T) {
	api, db, router := newTestAPI(t)

	//Create admin user
	u, pass := assets.InsertAdminUser(t, db)

	//Insert Project
	pkey := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, api.Cache, pkey, pkey)

	k := &sdk.ProjectKey{
		Name: "mykey",
		Type: "pgp",
	}

	vars := map[string]string{
		"projectKey": proj.Key,
	}

	jsonBody, _ := json.Marshal(k)
	body := bytes.NewBuffer(jsonBody)
	uri := router.GetRouteV2("POST", api.addKeyInProjectV2Handler, vars)
	req, err := http.NewRequest("POST", uri, body)
	test.NoError(t, err)
	assets.AuthentifyRequest(t, req, u, pass)

	// Do the request
	w := httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)
	require.Equal(t, 200, w.Code)

	var key sdk.ProjectKey
	test.NoError(t, json.Unmarshal(w.Body.Bytes(), &key))

	require.Equal(t, proj.ID, key.ProjectID)
}
