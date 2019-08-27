package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ovh/cds/engine/api/authentication"
	"github.com/ovh/cds/engine/api/authentication/builtin"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/sdk"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func AuthentififyBuiltinConsumer(t *testing.T, api *API, jwsToken string) string {
	uri := api.Router.GetRoute("POST", api.postAuthBuiltinSigninHandler, nil)
	test.NotEmpty(t, uri)
	btes, _ := json.Marshal(sdk.AuthConsumerSigninRequest{
		"token": jwsToken,
	})
	t.Logf("signin with jws : %s", jwsToken)
	req, err := http.NewRequest("POST", uri, bytes.NewBuffer(btes))
	require.NoError(t, err)

	//Do the request
	rec := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	assert.Equal(t, 200, rec.Code)

	var signinReponse sdk.AuthConsumerSigninResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &signinReponse))
	require.NotEmpty(t, signinReponse.Token, "session token should not be empty")
	require.NotNil(t, signinReponse.User, "user should not be nil")

	t.Logf("consumer authentified. jwt: %s", signinReponse.Token)

	assert.NotEmpty(t, rec.Header().Get("X-Api-Pub-Signing-Key"))

	return signinReponse.Token
}

func Test_postAuthBuiltinSigninHandler(t *testing.T) {
	api, _, _, end := newTestAPI(t)
	defer end()

	usr, _ := assets.InsertLambdaUser(t, api.mustDB(), &sdk.Group{Name: sdk.RandomString(5)})
	localConsumer, err := authentication.LoadConsumerByTypeAndUserID(context.TODO(), api.mustDB(), sdk.ConsumerLocal, usr.ID, authentication.LoadConsumerOptions.WithAuthentifiedUser)
	require.NoError(t, err)

	_, jws, err := builtin.NewConsumer(api.mustDB(), sdk.RandomString(10), sdk.RandomString(10), localConsumer, usr.GetGroupIDs(), Scope(sdk.AuthConsumerScopeProject))
	require.NoError(t, err)
	AuthentififyBuiltinConsumer(t, api, jws)
}

func Test_postAuthBuiltinRegenHandler(t *testing.T) {
	api, db, _, end := newTestAPI(t)
	defer end()

	u, jwtRaw := assets.InsertLambdaUser(t, db)
	localConsumer, err := authentication.LoadConsumerByTypeAndUserID(context.TODO(), db, sdk.ConsumerLocal, u.ID,
		authentication.LoadConsumerOptions.WithAuthentifiedUser)
	require.NoError(t, err)

	builtinConsumer, signinToken1, err := builtin.NewConsumer(db, sdk.RandomString(10), "", localConsumer, nil, []sdk.AuthConsumerScope{sdk.AuthConsumerScopeUser, sdk.AuthConsumerScopeAccessToken})
	require.NoError(t, err)
	session, err := authentication.NewSession(db, builtinConsumer, 5*time.Minute, false)
	require.NoError(t, err, "cannot create session")
	jwt2, err := authentication.NewSessionJWT(session)
	require.NoError(t, err, "cannot create jwt")

	uri := api.Router.GetRoute(http.MethodGet, api.getUserHandler, map[string]string{
		"permUsernamePublic": "me",
	})
	require.NotEmpty(t, uri)
	req := assets.NewJWTAuthentifiedRequest(t, jwtRaw, http.MethodGet, uri, nil)
	rec := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	req = assets.NewJWTAuthentifiedRequest(t, jwt2, http.MethodGet, uri, nil)
	rec = httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	// Wait 2 seconds before regen
	time.Sleep(2 * time.Second)

	uri = api.Router.GetRoute(http.MethodPost, api.postAuthBuiltinRegenHandler, nil)
	req = assets.NewJWTAuthentifiedRequest(t, jwt2, http.MethodPost, uri, sdk.AuthConsumerRegenRequest{
		ConsumerID:     builtinConsumer.ID,
		RevokeSessions: true,
	})
	rec = httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	var response sdk.AuthConsumerCreateResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))

	t.Logf("%+v", response)

	session, err = authentication.NewSession(db, builtinConsumer, 5*time.Minute, false)
	require.NoError(t, err)
	jwt3, err := authentication.NewSessionJWT(session)
	require.NoError(t, err)

	uri = api.Router.GetRoute(http.MethodGet, api.getUserHandler, map[string]string{
		"permUsernamePublic": "me",
	})

	// The new token should be ok
	req = assets.NewJWTAuthentifiedRequest(t, jwt3, http.MethodGet, uri, nil)
	rec = httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	// After the regen the old token should be invalidated because we choose to drop the sessions
	req = assets.NewJWTAuthentifiedRequest(t, jwt2, http.MethodGet, uri, nil)
	rec = httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusUnauthorized, rec.Code)

	// the old signing token from the builtin consumer should be invalidated
	uri = api.Router.GetRoute(http.MethodPost, api.postAuthBuiltinSigninHandler, nil)
	req = assets.NewRequest(t, "POST", uri, sdk.AuthConsumerSigninRequest{
		"token": signinToken1,
	})
	rec = httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusBadRequest, rec.Code)

	// the new signing token from the builtin consumer should be fine
	uri = api.Router.GetRoute(http.MethodPost, api.postAuthBuiltinSigninHandler, nil)
	req = assets.NewRequest(t, "POST", uri, sdk.AuthConsumerSigninRequest{
		"token": response.Token,
	})
	rec = httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

}
