package main

import (
	"testing"

	"github.com/gorilla/mux"
	"github.com/loopfz/gadgeto/iffy"

	"github.com/magiconair/properties/assert"
	"github.com/ovh/cds/engine/api/auth"
	"github.com/ovh/cds/engine/api/environment"
	"github.com/ovh/cds/engine/api/keys"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/sdk"
)

func Test_getKeysInEnvironmentHandler(t *testing.T) {
	db := test.SetupPG(t)

	router = newRouter(auth.TestLocalAuth(t), mux.NewRouter(), "/Test_getKeysInEnvironmentHandler")
	router.init()

	//Create admin user
	u, pass := assets.InsertAdminUser(db)

	//Create a fancy httptester
	tester := iffy.NewTester(t, router.mux)

	//Insert Project
	pkey := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, pkey, pkey, u)

	//Insert Application
	env := &sdk.Environment{
		Name:      sdk.RandomString(10),
		ProjectID: proj.ID,
	}
	if err := environment.InsertEnvironment(db, env); err != nil {
		t.Fatal(err)
	}

	k := &sdk.EnvironmentKey{
		Key: sdk.Key{
			Name: "mykey",
			Type: "pgp",
		},
		EnvironmentID: env.ID,
	}

	pub, priv, err := keys.GeneratePGPKeyPair(k.Name, u)
	if err != nil {
		t.Fatal(err)
	}
	k.Public = pub
	k.Private = priv

	if err := environment.InsertKey(db, k); err != nil {
		t.Fatal(err)
	}

	vars := map[string]string{
		"key": proj.Key,
		"permEnvironmentName": env.Name,
		"name":                k.Name,
	}

	route := router.getRoute("GET", getKeysInEnvironmentHandler, vars)
	headers := assets.AuthHeaders(t, u, pass)

	var keys []sdk.ApplicationKey
	tester.AddCall("Test_getKeysInEnvironmentHandler", "GET", route, nil).Headers(headers).Checkers(iffy.ExpectStatus(200), iffy.ExpectListLength(1), iffy.DumpResponse(t), iffy.UnmarshalResponse(&keys))
	tester.Run()
}

func Test_deleteKeyInEnvironmentHandler(t *testing.T) {
	db := test.SetupPG(t)

	router = newRouter(auth.TestLocalAuth(t), mux.NewRouter(), "/Test_deleteKeyInEnvironmentHandler")
	router.init()

	//Create admin user
	u, pass := assets.InsertAdminUser(db)

	//Create a fancy httptester
	tester := iffy.NewTester(t, router.mux)

	//Insert Project
	pkey := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, pkey, pkey, u)

	//Insert Application
	env := &sdk.Environment{
		Name:      sdk.RandomString(10),
		ProjectID: proj.ID,
	}
	if err := environment.InsertEnvironment(db, env); err != nil {
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

	if err := environment.InsertKey(db, k); err != nil {
		t.Fatal(err)
	}

	vars := map[string]string{
		"key": proj.Key,
		"permEnvironmentName": env.Name,
		"name":                k.Name,
	}

	route := router.getRoute("DELETE", deleteKeyInEnvironmentHandler, vars)
	headers := assets.AuthHeaders(t, u, pass)

	var keys []sdk.ApplicationKey
	tester.AddCall("Test_deleteKeyInEnvironmentHandler", "DELETE", route, nil).Headers(headers).Checkers(iffy.ExpectStatus(200), iffy.ExpectListLength(0), iffy.DumpResponse(t), iffy.UnmarshalResponse(&keys))
	tester.Run()
}

func Test_addKeyInEnvironmentHandler(t *testing.T) {
	db := test.SetupPG(t)

	router = newRouter(auth.TestLocalAuth(t), mux.NewRouter(), "/Test_addKeyInEnvironmentHandler")
	router.init()

	//Create admin user
	u, pass := assets.InsertAdminUser(db)

	//Create a fancy httptester
	tester := iffy.NewTester(t, router.mux)

	//Insert Project
	pkey := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, pkey, pkey, u)

	//Insert Environment
	env := &sdk.Environment{
		Name:      sdk.RandomString(10),
		ProjectID: proj.ID,
	}
	if err := environment.InsertEnvironment(db, env); err != nil {
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

	route := router.getRoute("POST", addKeyInEnvironmentHandler, vars)
	headers := assets.AuthHeaders(t, u, pass)

	var key sdk.EnvironmentKey
	tester.AddCall("Test_addKeyInEnvironmentHandler", "POST", route, k).Headers(headers).Checkers(iffy.ExpectStatus(200), iffy.UnmarshalResponse(&key))
	tester.Run()

	assert.Equal(t, env.ID, key.EnvironmentID)
}
