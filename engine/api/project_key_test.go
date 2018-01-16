package api

import (
	"testing"

	"github.com/loopfz/gadgeto/iffy"

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

	//Create a fancy httptester
	tester := iffy.NewTester(t, router.Mux)

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

	route := router.GetRoute("GET", api.getAllKeysProjectHandler, vars)
	headers := assets.AuthHeaders(t, u, pass)

	allkeys := struct {
		ProjectKeys     []sdk.ProjectKey     `json:"project_key"`
		ApplicationKeys []sdk.ApplicationKey `json:"application_key"`
		EnvironmentKeys []sdk.EnvironmentKey `json:"environment_key"`
	}{}
	tester.AddCall("Test_getAllKeysProjectHandler", "GET", route, nil).Headers(headers).Checkers(iffy.ExpectStatus(200), iffy.DumpResponse(t), iffy.UnmarshalResponse(&allkeys))
	tester.Run()

	assert.Equal(t, 1, len(allkeys.ProjectKeys))
	assert.Equal(t, 3, len(allkeys.ApplicationKeys))
	assert.Equal(t, 3, len(allkeys.EnvironmentKeys))
}

func Test_getKeysInProjectHandler(t *testing.T) {
	api, db, router := newTestAPI(t)

	//Create admin user
	u, pass := assets.InsertAdminUser(api.mustDB())

	//Create a fancy httptester
	tester := iffy.NewTester(t, router.Mux)

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

	route := router.GetRoute("GET", api.getKeysInProjectHandler, vars)
	headers := assets.AuthHeaders(t, u, pass)

	var keys []sdk.ProjectKey
	tester.AddCall("Test_getKeysInProjectHandler", "GET", route, nil).Headers(headers).Checkers(iffy.ExpectStatus(200), iffy.ExpectListLength(1), iffy.DumpResponse(t), iffy.UnmarshalResponse(&keys))
	tester.Run()
}

func Test_deleteKeyInProjectHandler(t *testing.T) {
	api, db, router := newTestAPI(t)

	//Create admin user
	u, pass := assets.InsertAdminUser(api.mustDB())

	//Create a fancy httptester
	tester := iffy.NewTester(t, router.Mux)

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

	route := router.GetRoute("DELETE", api.deleteKeyInProjectHandler, vars)
	headers := assets.AuthHeaders(t, u, pass)

	var keys []sdk.ProjectKey
	tester.AddCall("Test_deleteKeyInProjectHandler", "DELETE", route, nil).Headers(headers).Checkers(iffy.ExpectStatus(200), iffy.ExpectListLength(0), iffy.DumpResponse(t), iffy.UnmarshalResponse(&keys))
	tester.Run()
}

func Test_addKeyInProjectHandler(t *testing.T) {
	api, db, router := newTestAPI(t)

	//Create admin user
	u, pass := assets.InsertAdminUser(api.mustDB())

	//Create a fancy httptester
	tester := iffy.NewTester(t, router.Mux)

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

	route := router.GetRoute("POST", api.addKeyInProjectHandler, vars)
	headers := assets.AuthHeaders(t, u, pass)

	var key sdk.ProjectKey
	tester.AddCall("Test_addKeyInProjectHandler", "POST", route, k).Headers(headers).Checkers(iffy.ExpectStatus(200), iffy.UnmarshalResponse(&key))
	tester.Run()

	assert.Equal(t, proj.ID, key.ProjectID)
}
