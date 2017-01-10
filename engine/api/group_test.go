package main

import (
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"

	"github.com/ovh/cds/engine/api/bootstrap"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/sdk"
)

func Test_getPublicGroups(t *testing.T) {
	db := test.SetupPG(t, bootstrap.InitiliazeDB)

	router = &Router{test.LocalAuth(t), mux.NewRouter(), "/Test_getPublicGroups"}
	router.init()

	//Create group
	g := &sdk.Group{
		Name: test.RandomString(t, 10),
	}

	//Create user
	u, pass := test.InsertLambaUser(t, db, g)
	assert.NotZero(t, u)
	assert.NotZero(t, pass)

	//Prepare request
	uri := router.getRoute("GET", getPublicGroups, nil)
	test.NotEmpty(t, uri)

	req := test.NewAuthentifiedRequest(t, u, pass, "GET", uri, nil)

	//Do the request
	w := httptest.NewRecorder()
	router.mux.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)

	t.Logf("Body: %s", w.Body.String())
}
