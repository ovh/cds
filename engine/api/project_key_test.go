package main

import (
	"testing"

	"github.com/gorilla/mux"
	"github.com/loopfz/gadgeto/iffy"

	"github.com/magiconair/properties/assert"
	"github.com/ovh/cds/engine/api/auth"
	"github.com/ovh/cds/engine/api/keys"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/sdk"
)

func Test_getKeysInProjectHandler(t *testing.T) {
	db := test.SetupPG(t)

	router = newRouter(auth.TestLocalAuth(t), mux.NewRouter(), "/Test_getKeysInProjectHandler")
	router.init()

	//Create admin user
	u, pass := assets.InsertAdminUser(db)

	//Create a fancy httptester
	tester := iffy.NewTester(t, router.mux)

	//Insert Project
	pkey := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, pkey, pkey, u)

	k := &sdk.ProjectKey{
		Key: sdk.Key{
			Name: "mykey",
			Type: "pgp",
		},
		ProjectID: proj.ID,
	}

	pub, priv, err := keys.GeneratePGPKeyPair(k.Name, u)
	if err != nil {
		t.Fatal(err)
	}
	k.Public = pub
	k.Private = priv

	if err := project.InsertKey(db, k); err != nil {
		t.Fatal(err)
	}

	vars := map[string]string{
		"permProjectKey": proj.Key,
		"name":           k.Name,
	}

	route := router.getRoute("GET", getKeysInProjectHandler, vars)
	headers := assets.AuthHeaders(t, u, pass)

	var keys []sdk.ProjectKey
	tester.AddCall("Test_getKeysInProjectHandler", "GET", route, nil).Headers(headers).Checkers(iffy.ExpectStatus(200), iffy.ExpectListLength(1), iffy.DumpResponse(t), iffy.UnmarshalResponse(&keys))
	tester.Run()
}

func Test_deleteKeyInProjectHandler(t *testing.T) {
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

	k := &sdk.ProjectKey{
		Key: sdk.Key{
			Name:    "mykey",
			Type:    "pgp",
			Public:  "pub",
			Private: "priv",
		},
		ProjectID: proj.ID,
	}

	if err := project.InsertKey(db, k); err != nil {
		t.Fatal(err)
	}

	vars := map[string]string{
		"permProjectKey": proj.Key,
		"name":           k.Name,
	}

	route := router.getRoute("DELETE", deleteKeyInProjectHandler, vars)
	headers := assets.AuthHeaders(t, u, pass)

	var keys []sdk.ProjectKey
	tester.AddCall("Test_deleteKeyInProjectHandler", "DELETE", route, nil).Headers(headers).Checkers(iffy.ExpectStatus(200), iffy.ExpectListLength(0), iffy.DumpResponse(t), iffy.UnmarshalResponse(&keys))
	tester.Run()
}

func Test_addKeyInProjectHandler(t *testing.T) {
	db := test.SetupPG(t)

	router = newRouter(auth.TestLocalAuth(t), mux.NewRouter(), "/Test_addKeyInProjectHandler")
	router.init()

	//Create admin user
	u, pass := assets.InsertAdminUser(db)

	//Create a fancy httptester
	tester := iffy.NewTester(t, router.mux)

	//Insert Project
	pkey := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, pkey, pkey, u)

	k := &sdk.ProjectKey{
		Key: sdk.Key{
			Name: "mykey",
			Type: "pgp",
		},
	}

	vars := map[string]string{
		"permProjectKey": proj.Key,
	}

	route := router.getRoute("POST", addKeyInProjectHandler, vars)
	headers := assets.AuthHeaders(t, u, pass)

	var key sdk.ProjectKey
	tester.AddCall("Test_addKeyInProjectHandler", "POST", route, k).Headers(headers).Checkers(iffy.ExpectStatus(200), iffy.UnmarshalResponse(&key))
	tester.Run()

	assert.Equal(t, proj.ID, key.ProjectID)
}
