package main

import (
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"

	"github.com/ovh/cds/engine/api/auth"
	"github.com/ovh/cds/engine/api/bootstrap"
	"github.com/ovh/cds/engine/api/sessionstore"
	"github.com/ovh/cds/engine/api/testwithdb"
	"github.com/ovh/cds/sdk"
)

func Test_getPublicGroups(t *testing.T) {
	if testwithdb.DBDriver == "" {
		t.SkipNow()
		return
	}
	db, err := testwithdb.SetupPG(t, bootstrap.InitiliazeDB)
	assert.NoError(t, err)
	if db == nil {
		t.FailNow()
	}

	authDriver, _ := auth.GetDriver("local", nil, sessionstore.Options{Mode: "local", TTL: 30})
	router = &Router{authDriver, mux.NewRouter(), "/Test_getPublicGroups"}
	router.init()

	//Create group
	g := &sdk.Group{
		Name: testwithdb.RandomString(t, 10),
	}

	//Create user
	u, pass, err := testwithdb.InsertLambaUser(t, db, g)
	assert.NoError(t, err)
	assert.NotZero(t, u)
	assert.NotZero(t, pass)

	//Prepare request
	uri := router.getRoute("GET", getPublicGroups, nil)
	if uri == "" {
		t.FailNow()
	}
	req := testwithdb.NewAuthentifiedRequest(t, u, pass, "GET", uri, nil)

	//Do the request
	w := httptest.NewRecorder()
	router.mux.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)

	t.Logf("Body: %s", w.Body.String())
}
