package api

import (
	"testing"

	"github.com/loopfz/gadgeto/iffy"

	"github.com/ovh/cds/engine/api/keys"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/sdk"
	"github.com/stretchr/testify/assert"
)

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

	kid, pub, priv, err := keys.GeneratePGPKeyPair(k.Name)
	if err != nil {
		t.Fatal(err)
	}
	k.Public = pub
	k.Private = priv
	k.KeyID = kid

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
