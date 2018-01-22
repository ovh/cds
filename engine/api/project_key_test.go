package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/environment"
	"github.com/ovh/cds/engine/api/keys"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/sdk"
	"github.com/stretchr/testify/assert"
)

func Test_getAllKeysProjectHandler(t *testing.T) {
	api, db, router := newTestAPI(t)

	//Create admin user
	u, pass := assets.InsertAdminUser(api.mustDB())

	//Insert Project
	pkey := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, api.Cache, pkey, pkey, u)

	k := &sdk.ProjectKey{
		Key: sdk.Key{
			Name: "mykey",
			Type: "pgp",
		},
		ProjectID: proj.ID,
	}
	test.NoError(t, project.InsertKey(api.mustDB(), k))

	app := sdk.Application{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       sdk.RandomString(10),
	}

	test.NoError(t, application.Insert(db, api.Cache, proj, &app, u))

	appk1 := &sdk.ApplicationKey{
		Key: sdk.Key{
			Name: "appK1",
			Type: "ssh",
		},
		ApplicationID: app.ID,
	}
	appk2 := &sdk.ApplicationKey{
		Key: sdk.Key{
			Name: "appK2",
			Type: "ssh",
		},
		ApplicationID: app.ID,
	}
	test.NoError(t, application.InsertKey(db, appk1))
	test.NoError(t, application.InsertKey(db, appk2))

	app2 := sdk.Application{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       sdk.RandomString(10),
	}

	test.NoError(t, application.Insert(db, api.Cache, proj, &app2, u))

	app2k1 := &sdk.ApplicationKey{
		Key: sdk.Key{
			Name: "appK1",
			Type: "ssh",
		},
		ApplicationID: app2.ID,
	}
	app2k2 := &sdk.ApplicationKey{
		Key: sdk.Key{
			Name: "appK2",
			Type: "pgp",
		},
		ApplicationID: app2.ID,
	}
	test.NoError(t, application.InsertKey(db, app2k1))
	test.NoError(t, application.InsertKey(db, app2k2))

	vars := map[string]string{
		"permProjectKey": proj.Key,
		"name":           k.Name,
	}

	env := sdk.Environment{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       sdk.RandomString(10),
	}
	test.NoError(t, environment.InsertEnvironment(db, &env))

	envk1 := &sdk.EnvironmentKey{
		Key: sdk.Key{
			Name: "envK1",
			Type: "ssh",
		},
		EnvironmentID: env.ID,
	}
	envk2 := &sdk.EnvironmentKey{
		Key: sdk.Key{
			Name: "envK2",
			Type: "ssh",
		},
		EnvironmentID: env.ID,
	}
	test.NoError(t, environment.InsertKey(db, envk1))
	test.NoError(t, environment.InsertKey(db, envk2))

	env2 := sdk.Environment{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       sdk.RandomString(10),
	}
	test.NoError(t, environment.InsertEnvironment(db, &env2))

	env2k1 := &sdk.EnvironmentKey{
		Key: sdk.Key{
			Name: "envK1",
			Type: "ssh",
		},
		EnvironmentID: env2.ID,
	}
	env2k2 := &sdk.EnvironmentKey{
		Key: sdk.Key{
			Name: "envK2",
			Type: "spgpsh",
		},
		EnvironmentID: env2.ID,
	}
	test.NoError(t, environment.InsertKey(db, env2k1))
	test.NoError(t, environment.InsertKey(db, env2k2))

	allkeys := struct {
		ProjectKeys     []sdk.ProjectKey     `json:"project_key"`
		ApplicationKeys []sdk.ApplicationKey `json:"application_key"`
		EnvironmentKeys []sdk.EnvironmentKey `json:"environment_key"`
	}{}

	uri := router.GetRoute("GET", api.getAllKeysProjectHandler, vars)
	req, err := http.NewRequest("GET", uri, nil)
	test.NoError(t, err)
	assets.AuthentifyRequest(t, req, u, pass)

	// Do the request
	w := httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)

	test.NoError(t, json.Unmarshal(w.Body.Bytes(), &allkeys))

	assert.Equal(t, 1, len(allkeys.ProjectKeys))
	assert.Equal(t, 3, len(allkeys.ApplicationKeys))
	assert.Equal(t, 3, len(allkeys.EnvironmentKeys))
}

func Test_getKeysInProjectHandler(t *testing.T) {
	api, db, router := newTestAPI(t)

	//Create admin user
	u, pass := assets.InsertAdminUser(api.mustDB())

	//Insert Project
	pkey := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, api.Cache, pkey, pkey, u)

	k := &sdk.ProjectKey{
		Key: sdk.Key{
			Name: "mykey",
			Type: "pgp",
		},
		ProjectID: proj.ID,
	}

	kPGP, err := keys.GeneratePGPKeyPair(k.Name)
	if err != nil {
		t.Fatal(err)
	}
	k.Key = kPGP

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
	api, db, router := newTestAPI(t)

	//Create admin user
	u, pass := assets.InsertAdminUser(api.mustDB())

	//Insert Project
	pkey := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, api.Cache, pkey, pkey, u)

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
	api, db, router := newTestAPI(t)

	//Create admin user
	u, pass := assets.InsertAdminUser(api.mustDB())

	//Insert Project
	pkey := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, api.Cache, pkey, pkey, u)

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
