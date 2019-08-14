package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/sdk"
)

func Test_getUsers(t *testing.T) {
	api, db, _, end := newTestAPI(t)
	defer end()

	expected, jwtRaw := assets.InsertLambdaUser(db)

	uri := api.Router.GetRoute(http.MethodGet, api.getUsersHandler, nil)
	require.NotEmpty(t, uri)
	req := assets.NewJWTAuthentifiedRequest(t, jwtRaw, http.MethodGet, uri, nil)
	rec := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	require.Equal(t, 200, rec.Code)

	var us []sdk.AuthentifiedUser
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &us))
	require.True(t, len(us) >= 1)

	var result *sdk.AuthentifiedUser
	for _, u := range us {
		if expected.ID == u.ID {
			result = &u
			break
		}
	}
	require.NotNil(t, result, "user should be in the list of all users")
	assert.Equal(t, expected.Username, result.Username)
}

func Test_getUser(t *testing.T) {
	api, db, _, end := newTestAPI(t)
	defer end()

	expected, jwtRaw := assets.InsertLambdaUser(db)

	uri := api.Router.GetRoute(http.MethodGet, api.getUserHandler, map[string]string{
		"permUsernamePublic": expected.Username,
	})
	require.NotEmpty(t, uri)
	req := assets.NewJWTAuthentifiedRequest(t, jwtRaw, http.MethodGet, uri, nil)
	rec := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	require.Equal(t, 200, rec.Code)

	var u sdk.AuthentifiedUser
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &u))
	require.Equal(t, expected.ID, u.ID)
}
