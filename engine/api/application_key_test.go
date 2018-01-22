package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/keys"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/sdk"
)

func Test_getKeysInApplicationHandler(t *testing.T) {
	api, db, router := newTestAPI(t)

	//Create admin user
	u, pass := assets.InsertAdminUser(api.mustDB())

	//Insert Project
	pkey := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, api.Cache, pkey, pkey, u)

	//Insert Application
	app := &sdk.Application{
		Name: sdk.RandomString(10),
	}
	if err := application.Insert(api.mustDB(), api.Cache, proj, app, u); err != nil {
		t.Fatal(err)
	}

	k := &sdk.ApplicationKey{
		Key: sdk.Key{
			Name: "mykey",
			Type: "pgp",
		},
		ApplicationID: app.ID,
	}

	pgpK, err := keys.GeneratePGPKeyPair(k.Name)
	if err != nil {
		t.Fatal(err)
	}

	k.Key = pgpK

	if err := application.InsertKey(api.mustDB(), k); err != nil {
		t.Fatal(err)
	}

	vars := map[string]string{
		"key": proj.Key,
		"permApplicationName": app.Name,
		"name":                k.Name,
	}

	uri := router.GetRoute("GET", api.getKeysInApplicationHandler, vars)
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

func Test_deleteKeyInApplicationHandler(t *testing.T) {
	api, db, router := newTestAPI(t)

	//Create admin user
	u, pass := assets.InsertAdminUser(api.mustDB())

	//Insert Project
	pkey := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, api.Cache, pkey, pkey, u)

	//Insert Application
	app := &sdk.Application{
		Name: sdk.RandomString(10),
	}
	if err := application.Insert(api.mustDB(), api.Cache, proj, app, u); err != nil {
		t.Fatal(err)
	}

	k := &sdk.ApplicationKey{
		Key: sdk.Key{
			Name:    "mykey",
			Type:    "pgp",
			Public:  "pub",
			Private: "priv",
		},
		ApplicationID: app.ID,
	}

	if err := application.InsertKey(api.mustDB(), k); err != nil {
		t.Fatal(err)
	}

	vars := map[string]string{
		"key": proj.Key,
		"permApplicationName": app.Name,
		"name":                k.Name,
	}

	uri := router.GetRoute("DELETE", api.deleteKeyInApplicationHandler, vars)

	req, err := http.NewRequest("DELETE", uri, nil)
	test.NoError(t, err)
	assets.AuthentifyRequest(t, req, u, pass)

	// Do the request
	w := httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)

	var keys []sdk.ApplicationKey
	test.NoError(t, json.Unmarshal(w.Body.Bytes(), &keys))
	assert.Equal(t, len(keys), 0)
}

func Test_addKeyInApplicationHandler(t *testing.T) {
	api, db, router := newTestAPI(t)

	//Create admin user
	u, pass := assets.InsertAdminUser(api.mustDB())

	//Insert Project
	pkey := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, api.Cache, pkey, pkey, u)

	//Insert Application
	app := &sdk.Application{
		Name: sdk.RandomString(10),
	}
	if err := application.Insert(api.mustDB(), api.Cache, proj, app, u); err != nil {
		t.Fatal(err)
	}

	k := &sdk.ApplicationKey{
		Key: sdk.Key{
			Name: "mykey",
			Type: "pgp",
		},
	}

	vars := map[string]string{
		"key": proj.Key,
		"permApplicationName": app.Name,
	}

	jsonBody, _ := json.Marshal(k)
	body := bytes.NewBuffer(jsonBody)
	uri := router.GetRoute("POST", api.addKeyInApplicationHandler, vars)
	req, err := http.NewRequest("POST", uri, body)
	test.NoError(t, err)
	assets.AuthentifyRequest(t, req, u, pass)

	// Do the request
	w := httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)

	var key sdk.ApplicationKey
	test.NoError(t, json.Unmarshal(w.Body.Bytes(), &key))
	assert.Equal(t, app.ID, key.ApplicationID)
}
