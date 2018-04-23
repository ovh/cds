package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ovh/cds/engine/api/environment"
	"github.com/ovh/cds/engine/api/keys"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/sdk"
	"github.com/stretchr/testify/assert"
)

func Test_getKeysInEnvironmentHandler(t *testing.T) {
	api, db, router := newTestAPI(t)

	//Create admin user
	u, pass := assets.InsertAdminUser(api.mustDB(context.Background()))

	//Insert Project
	pkey := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, api.Cache, pkey, pkey, u)

	//Insert Application
	env := &sdk.Environment{
		Name:      sdk.RandomString(10),
		ProjectID: proj.ID,
	}
	if err := environment.InsertEnvironment(api.mustDB(context.Background()), env); err != nil {
		t.Fatal(err)
	}

	k := &sdk.EnvironmentKey{
		Key: sdk.Key{
			Name: "mykey",
			Type: "pgp",
		},
		EnvironmentID: env.ID,
	}

	kpgp, err := keys.GeneratePGPKeyPair(k.Name)
	if err != nil {
		t.Fatal(err)
	}

	k.Public = kpgp.Public
	k.Private = kpgp.Private
	k.KeyID = kpgp.KeyID

	if err := environment.InsertKey(api.mustDB(context.Background()), k); err != nil {
		t.Fatal(err)
	}

	vars := map[string]string{
		"key": proj.Key,
		"permEnvironmentName": env.Name,
		"name":                k.Name,
	}

	uri := router.GetRoute("GET", api.getKeysInEnvironmentHandler, vars)
	req, err := http.NewRequest("GET", uri, nil)
	test.NoError(t, err)
	assets.AuthentifyRequest(t, req, u, pass)

	// Do the request
	w := httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)

	var keys []sdk.ApplicationKey
	test.NoError(t, json.Unmarshal(w.Body.Bytes(), &keys))
	assert.Equal(t, len(keys), 1)

}

func Test_deleteKeyInEnvironmentHandler(t *testing.T) {
	api, db, router := newTestAPI(t)

	//Create admin user
	u, pass := assets.InsertAdminUser(api.mustDB(context.Background()))

	//Insert Project
	pkey := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, api.Cache, pkey, pkey, u)

	//Insert Application
	env := &sdk.Environment{
		Name:      sdk.RandomString(10),
		ProjectID: proj.ID,
	}
	if err := environment.InsertEnvironment(api.mustDB(context.Background()), env); err != nil {
		t.Fatal(err)
	}

	k := &sdk.EnvironmentKey{
		Key: sdk.Key{
			Name:    "mykey",
			Type:    "pgp",
			Public:  "pub",
			Private: "priv",
		},
		EnvironmentID: env.ID,
	}

	if err := environment.InsertKey(api.mustDB(context.Background()), k); err != nil {
		t.Fatal(err)
	}

	vars := map[string]string{
		"key": proj.Key,
		"permEnvironmentName": env.Name,
		"name":                k.Name,
	}

	uri := router.GetRoute("DELETE", api.deleteKeyInEnvironmentHandler, vars)
	req, err := http.NewRequest("DELETE", uri, nil)
	test.NoError(t, err)
	assets.AuthentifyRequest(t, req, u, pass)

	// Do the request
	w := httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)

	var keys []sdk.EnvironmentKey
	test.NoError(t, json.Unmarshal(w.Body.Bytes(), &keys))
	assert.Equal(t, len(keys), 0)

}

func Test_addKeyInEnvironmentHandler(t *testing.T) {
	api, db, router := newTestAPI(t)

	//Create admin user
	u, pass := assets.InsertAdminUser(api.mustDB(context.Background()))

	//Insert Project
	pkey := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, api.Cache, pkey, pkey, u)

	//Insert Environment
	env := &sdk.Environment{
		Name:      sdk.RandomString(10),
		ProjectID: proj.ID,
	}
	if err := environment.InsertEnvironment(api.mustDB(context.Background()), env); err != nil {
		t.Fatal(err)
	}

	k := &sdk.EnvironmentKey{
		Key: sdk.Key{
			Name: "mykey",
			Type: "pgp",
		},
	}

	vars := map[string]string{
		"key": proj.Key,
		"permEnvironmentName": env.Name,
	}

	jsonBody, _ := json.Marshal(k)
	body := bytes.NewBuffer(jsonBody)

	uri := router.GetRoute("POST", api.addKeyInEnvironmentHandler, vars)
	req, err := http.NewRequest("POST", uri, body)
	test.NoError(t, err)
	assets.AuthentifyRequest(t, req, u, pass)

	// Do the request
	w := httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)

	var key sdk.EnvironmentKey
	test.NoError(t, json.Unmarshal(w.Body.Bytes(), &key))

	assert.Equal(t, env.ID, key.EnvironmentID)
}
