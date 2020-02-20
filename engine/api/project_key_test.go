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
	"github.com/stretchr/testify/assert"
)

func Test_getKeysInProjectHandler(t *testing.T) {
	api, db, router, end := newTestAPI(t)
	defer end()

	//Create admin user
	u, pass := assets.InsertAdminUser(t, api.mustDB())

	//Insert Project
	pkey := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, api.Cache, pkey, pkey)

	k := &sdk.ProjectKey{
		Key: sdk.Key{
			Name: "mykey",
			Type: "pgp",
		},
		ProjectID: proj.ID,
	}

	kpgp, err := keys.GeneratePGPKeyPair(k.Name)
	if err != nil {
		t.Fatal(err)
	}
	k.Key = kpgp

	if err := project.InsertKey(api.mustDB(), k); err != nil {
		t.Fatal(err)
	}

	vars := map[string]string{
		"permProjectKey": proj.Key,
		"name":           k.Name,
	}

	uri := router.GetRoute("GET", api.getKeysInProjectHandler, vars)
	req, err := http.NewRequest("GET", uri, nil)
	test.NoError(t, err)
	assets.AuthentifyRequest(t, req, u, pass)

	// Do the request
	w := httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)

	var keys []sdk.ProjectKey
	test.NoError(t, json.Unmarshal(w.Body.Bytes(), &keys))
	assert.Equal(t, len(keys), 1)
}

func Test_deleteKeyInProjectHandler(t *testing.T) {
	api, db, router, end := newTestAPI(t)
	defer end()

	//Create admin user
	u, pass := assets.InsertAdminUser(t, api.mustDB())

	//Insert Project
	pkey := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, api.Cache, pkey, pkey)

	k := &sdk.ProjectKey{
		Key: sdk.Key{
			Name:    "mykey",
			Type:    "pgp",
			Public:  "pub",
			Private: "priv",
		},
		ProjectID: proj.ID,
	}

	if err := project.InsertKey(api.mustDB(), k); err != nil {
		t.Fatal(err)
	}

	vars := map[string]string{
		"permProjectKey": proj.Key,
		"name":           k.Name,
	}

	uri := router.GetRoute("DELETE", api.deleteKeyInProjectHandler, vars)
	req, err := http.NewRequest("DELETE", uri, nil)
	test.NoError(t, err)
	assets.AuthentifyRequest(t, req, u, pass)

	// Do the request
	w := httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)

	var keys []sdk.ProjectKey
	test.NoError(t, json.Unmarshal(w.Body.Bytes(), &keys))
	assert.Equal(t, len(keys), 0)
}

func Test_addKeyInProjectHandler(t *testing.T) {
	api, db, router, end := newTestAPI(t)
	defer end()

	//Create admin user
	u, pass := assets.InsertAdminUser(t, api.mustDB())

	//Insert Project
	pkey := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, api.Cache, pkey, pkey)

	k := &sdk.ProjectKey{
		Key: sdk.Key{
			Name: "mykey",
			Type: "pgp",
		},
	}

	vars := map[string]string{
		"permProjectKey": proj.Key,
	}

	jsonBody, _ := json.Marshal(k)
	body := bytes.NewBuffer(jsonBody)
	uri := router.GetRoute("POST", api.addKeyInProjectHandler, vars)
	req, err := http.NewRequest("POST", uri, body)
	test.NoError(t, err)
	assets.AuthentifyRequest(t, req, u, pass)

	// Do the request
	w := httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)

	var key sdk.ProjectKey
	test.NoError(t, json.Unmarshal(w.Body.Bytes(), &key))

	assert.Equal(t, proj.ID, key.ProjectID)
}
