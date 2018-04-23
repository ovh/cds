package api

import (
	"context"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ovh/cds/engine/api/bootstrap"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/sdk"
)

func Test_getPublicGroupsHandler(t *testing.T) {
	api, _, router := newTestAPI(t, bootstrap.InitiliazeDB)

	//Create group
	g := &sdk.Group{
		Name: sdk.RandomString(10),
	}

	//Create user
	u, pass := assets.InsertLambdaUser(api.mustDB(context.Background()), g)
	assert.NotZero(t, u)
	assert.NotZero(t, pass)

	//Prepare request
	uri := router.GetRoute("GET", api.getPublicGroupsHandler, nil)
	test.NotEmpty(t, uri)

	req := assets.NewAuthentifiedRequest(t, u, pass, "GET", uri, nil)

	//Do the request
	w := httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)

	t.Logf("Body: %s", w.Body.String())
}
