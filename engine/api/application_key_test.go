package main

import (
	"testing"

	"github.com/gorilla/mux"
	"github.com/loopfz/gadgeto/iffy"

	"github.com/magiconair/properties/assert"
	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/auth"
	"github.com/ovh/cds/engine/api/keys"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/sdk"
)

func Test_getKeysInApplicationHandler(t *testing.T) {
	db := test.SetupPG(t)

	router = newRouter(auth.TestLocalAuth(t), mux.NewRouter(), "/Test_getKeysInApplicationHandler")
	router.init()

	//Create admin user
	u, pass := assets.InsertAdminUser(db)

	//Create a fancy httptester
	tester := iffy.NewTester(t, router.mux)

	//Insert Project
	pkey := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, pkey, pkey, u)

	//Insert Application
	app := &sdk.Application{
		Name: sdk.RandomString(10),
	}
	if err := application.Insert(db, proj, app, u); err != nil {
		t.Fatal(err)
	}

	k := &sdk.ApplicationKey{
		Key: sdk.Key{
			Name: "mykey",
			Type: "pgp",
		},
		ApplicationID: app.ID,
	}

	pub, priv, err := keys.GeneratePGPKeyPair(k.Name, u)
	if err != nil {
		t.Fatal(err)
	}
	k.Public = pub
	k.Private = priv

	if err := application.InsertKey(db, k); err != nil {
		t.Fatal(err)
	}

	vars := map[string]string{
		"key": proj.Key,
		"permApplicationName": app.Name,
		"name":                k.Name,
	}

	route := router.getRoute("GET", getKeysInApplicationHandler, vars)
	headers := assets.AuthHeaders(t, u, pass)

	var keys []sdk.ApplicationKey
	tester.AddCall("Test_getKeysInApplicationHandler", "GET", route, nil).Headers(headers).Checkers(iffy.ExpectStatus(200), iffy.ExpectListLength(1), iffy.DumpResponse(t), iffy.UnmarshalResponse(&keys))
	tester.Run()
}

func Test_deleteKeyInApplicationHandler(t *testing.T) {
	db := test.SetupPG(t)

	router = newRouter(auth.TestLocalAuth(t), mux.NewRouter(), "/Test_deleteKeyInProjectHandler")
	router.init()

	//Create admin user
	u, pass := assets.InsertAdminUser(db)

	//Create a fancy httptester
	tester := iffy.NewTester(t, router.mux)

	//Insert Project
	pkey := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, pkey, pkey, u)

	//Insert Application
	app := &sdk.Application{
		Name: sdk.RandomString(10),
	}
	if err := application.Insert(db, proj, app, u); err != nil {
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

	if err := application.InsertKey(db, k); err != nil {
		t.Fatal(err)
	}

	vars := map[string]string{
		"key": proj.Key,
		"permApplicationName": app.Name,
		"name":                k.Name,
	}

	route := router.getRoute("DELETE", deleteKeyInApplicationHandler, vars)
	headers := assets.AuthHeaders(t, u, pass)

	var keys []sdk.ApplicationKey
	tester.AddCall("Test_deleteKeyInApplicationHandler", "DELETE", route, nil).Headers(headers).Checkers(iffy.ExpectStatus(200), iffy.ExpectListLength(0), iffy.DumpResponse(t), iffy.UnmarshalResponse(&keys))
	tester.Run()
}

func Test_addKeyInApplicationHandler(t *testing.T) {
	db := test.SetupPG(t)

	router = newRouter(auth.TestLocalAuth(t), mux.NewRouter(), "/Test_addKeyInApplicationHandler")
	router.init()

	//Create admin user
	u, pass := assets.InsertAdminUser(db)

	//Create a fancy httptester
	tester := iffy.NewTester(t, router.mux)

	//Insert Project
	pkey := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, pkey, pkey, u)

	//Insert Application
	app := &sdk.Application{
		Name: sdk.RandomString(10),
	}
	if err := application.Insert(db, proj, app, u); err != nil {
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

	route := router.getRoute("POST", addKeyInApplicationHandler, vars)
	headers := assets.AuthHeaders(t, u, pass)

	var key sdk.ApplicationKey
	tester.AddCall("Test_addKeyInApplicationHandler", "POST", route, k).Headers(headers).Checkers(iffy.ExpectStatus(200), iffy.UnmarshalResponse(&key))
	tester.Run()

	assert.Equal(t, app.ID, key.ApplicationID)
}
