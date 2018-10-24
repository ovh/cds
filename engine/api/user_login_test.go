package api

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/sdk"
	"github.com/stretchr/testify/assert"
)

// Create an admin user and return the jwt token
func newAdminUser(t *testing.T, api *API) (*sdk.User, string) {
	db := api.mustDB()
	u, p := assets.InsertAdminUser(db)

	// === Ask for the authentication method
	uri := api.Router.GetRoute("GET", api.getLoginUserHandler, nil)
	test.NotEmpty(t, uri)
	req := assets.NewRequest(t, "GET", uri, nil)
	w := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)
	if t.Failed() {
		t.FailNow()
	}

	var userLoginMethod sdk.UserLoginDriverResponse
	test.NoError(t, json.Unmarshal(w.Body.Bytes(), &userLoginMethod))
	assert.True(t, userLoginMethod.Local.Available)
	if t.Failed() {
		t.FailNow()
	}

	// === Call for authentify localy
	uri = api.Router.GetRoute("POST", api.postLoginUserHandler, nil)
	test.NotEmpty(t, uri)
	data := sdk.UserLoginRequest{
		Username: u.Username,
		Password: p,
	}

	req = assets.NewRequest(t, "POST", uri, data)

	//Do the request
	w = httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	if t.Failed() {
		t.FailNow()
	}

	var userLoginResponse sdk.UserAPIResponse
	test.NoError(t, json.Unmarshal(w.Body.Bytes(), &userLoginResponse))

	return &userLoginResponse.User, userLoginResponse.Token
}

// Create an lambda user and return the jwt token
func newLambdaUser(t *testing.T, api *API, groups ...*sdk.Group) (*sdk.User, string) {
	u, p := newLambdaUser(t, api, groups...)

	// === Ask for the authentication method
	uri := api.Router.GetRoute("GET", api.getLoginUserHandler, nil)
	test.NotEmpty(t, uri)
	req := assets.NewRequest(t, "GET", uri, nil)
	w := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)
	if t.Failed() {
		t.FailNow()
	}

	var userLoginMethod sdk.UserLoginDriverResponse
	test.NoError(t, json.Unmarshal(w.Body.Bytes(), &userLoginMethod))
	assert.True(t, userLoginMethod.Local.Available)
	if t.Failed() {
		t.FailNow()
	}

	// === Call for authentify localy
	uri = api.Router.GetRoute("POST", api.postLoginUserHandler, nil)
	test.NotEmpty(t, uri)
	data := sdk.UserLoginRequest{
		Username: u.Username,
		Password: p,
	}

	req = assets.NewRequest(t, "POST", uri, data)

	//Do the request
	w = httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	if t.Failed() {
		t.FailNow()
	}

	var userLoginResponse sdk.UserAPIResponse
	test.NoError(t, json.Unmarshal(w.Body.Bytes(), &userLoginResponse))

	return &userLoginResponse.User, userLoginResponse.Token
}

func Test_loginAdminUserHandler(t *testing.T) {
	api, _, _, end := newTestAPI(t)
	defer end()
	_, token := newAdminUser(t, api)

	// === Call an authentified method
	uri := api.Router.GetRoute("GET", api.getProjectsHandler, nil)
	test.NotEmpty(t, uri)
	req := assets.NewRequest(t, "GET", uri, nil)
	assets.AuthentifyRequestWithJWT(t, req, token)

	//Do the request
	w := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w, req)
	t.Logf("Response: %s", w.Body.String())

	assert.Equal(t, 200, w.Code)
}

func Test_loginLambdaUserHandler(t *testing.T) {
	api, _, _, end := newTestAPI(t)
	defer end()
	u, token := newLambdaUser(t, api)

	t.Logf("u=%+v", u)

	// === Call an authentified method
	uri := api.Router.GetRoute("GET", api.getProjectsHandler, nil)
	test.NotEmpty(t, uri)
	req := assets.NewRequest(t, "GET", uri, nil)
	assets.AuthentifyRequestWithJWT(t, req, token)

	//Do the request
	w := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w, req)
	t.Logf("Response: %s", w.Body.String())

	assert.Equal(t, 200, w.Code)
}
