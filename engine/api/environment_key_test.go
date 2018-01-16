package api

import (
	"testing"

	"github.com/loopfz/gadgeto/iffy"

	"github.com/ovh/cds/engine/api/environment"
	"github.com/ovh/cds/engine/api/keys"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/sdk"
	"github.com/stretchr/testify/assert"
)

func Test_getKeysInEnvironmentHandler(t *testing.T) {
	api, db, router := newTestAPI(t)

	//Create admin user
	u, pass := assets.InsertAdminUser(api.mustDB())

	//Create a fancy httptester
	tester := iffy.NewTester(t, router.Mux)

	//Insert Project
	pkey := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, api.Cache, pkey, pkey, u)

	//Insert Application
	env := &sdk.Environment{
		Name:      sdk.RandomString(10),
		ProjectID: proj.ID,
	}
	if err := environment.InsertEnvironment(api.mustDB(), env); err != nil {
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

	if err := environment.InsertKey(api.mustDB(), k); err != nil {
		t.Fatal(err)
	}

	vars := map[string]string{
		"key": proj.Key,
		"permEnvironmentName": env.Name,
		"name":                k.Name,
	}

	route := router.GetRoute("GET", api.getKeysInEnvironmentHandler, vars)
	headers := assets.AuthHeaders(t, u, pass)

	var keys []sdk.ApplicationKey
	tester.AddCall("Test_getKeysInEnvironmentHandler", "GET", route, nil).Headers(headers).Checkers(iffy.ExpectStatus(200), iffy.ExpectListLength(1), iffy.DumpResponse(t), iffy.UnmarshalResponse(&keys))
	tester.Run()
}

func Test_deleteKeyInEnvironmentHandler(t *testing.T) {
	api, db, router := newTestAPI(t)

	//Create admin user
	u, pass := assets.InsertAdminUser(api.mustDB())

	//Create a fancy httptester
	tester := iffy.NewTester(t, router.Mux)

	//Insert Project
	pkey := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, api.Cache, pkey, pkey, u)

	//Insert Application
	env := &sdk.Environment{
		Name:      sdk.RandomString(10),
		ProjectID: proj.ID,
	}
	if err := environment.InsertEnvironment(api.mustDB(), env); err != nil {
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

	if err := environment.InsertKey(api.mustDB(), k); err != nil {
		t.Fatal(err)
	}

	vars := map[string]string{
		"key": proj.Key,
		"permEnvironmentName": env.Name,
		"name":                k.Name,
	}

	route := router.GetRoute("DELETE", api.deleteKeyInEnvironmentHandler, vars)
	headers := assets.AuthHeaders(t, u, pass)

	var keys []sdk.ApplicationKey
	tester.AddCall("Test_deleteKeyInEnvironmentHandler", "DELETE", route, nil).Headers(headers).Checkers(iffy.ExpectStatus(200), iffy.ExpectListLength(0), iffy.DumpResponse(t), iffy.UnmarshalResponse(&keys))
	tester.Run()
}

func Test_addKeyInEnvironmentHandler(t *testing.T) {
	api, db, router := newTestAPI(t)

	//Create admin user
	u, pass := assets.InsertAdminUser(api.mustDB())

	//Create a fancy httptester
	tester := iffy.NewTester(t, router.Mux)

	//Insert Project
	pkey := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, api.Cache, pkey, pkey, u)

	//Insert Environment
	env := &sdk.Environment{
		Name:      sdk.RandomString(10),
		ProjectID: proj.ID,
	}
	if err := environment.InsertEnvironment(api.mustDB(), env); err != nil {
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

	route := router.GetRoute("POST", api.addKeyInEnvironmentHandler, vars)
	headers := assets.AuthHeaders(t, u, pass)

	var key sdk.EnvironmentKey
	tester.AddCall("Test_addKeyInEnvironmentHandler", "POST", route, k).Headers(headers).Checkers(iffy.ExpectStatus(200), iffy.UnmarshalResponse(&key))
	tester.Run()

	assert.Equal(t, env.ID, key.EnvironmentID)
}
