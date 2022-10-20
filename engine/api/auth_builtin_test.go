package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ovh/cds/engine/api/authentication"
	"github.com/ovh/cds/engine/api/authentication/builtin"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/sdk"

	"github.com/stretchr/testify/require"
)

func AuthentififyBuiltinConsumer(t *testing.T, api *API, jwsToken string, srv *sdk.Service) string {
	uri := api.Router.GetRoute("POST", api.postAuthBuiltinSigninHandler, nil)
	require.NotEmpty(t, uri)
	reqSignin := sdk.AuthConsumerSigninRequest{"token": jwsToken}
	if srv != nil {
		reqSignin["service"] = srv
	}
	btes, err := json.Marshal(reqSignin)
	require.NoError(t, err)

	t.Logf("signin with jws : %s", jwsToken)

	req, err := http.NewRequest("POST", uri, bytes.NewBuffer(btes))
	require.NoError(t, err)
	rec := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	require.Equal(t, 200, rec.Code)

	var signinReponse sdk.AuthConsumerSigninResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &signinReponse))
	require.NotEmpty(t, signinReponse.Token, "session token should not be empty")
	require.NotNil(t, signinReponse.User, "user should not be nil")

	t.Logf("consumer authentified. jwt: %s", signinReponse.Token)

	require.NotEmpty(t, rec.Header().Get("X-Api-Pub-Signing-Key"))

	return signinReponse.Token
}

func Test_postAuthBuiltinSigninHandler(t *testing.T) {
	api, db, _ := newTestAPI(t)

	usr, _ := assets.InsertLambdaUser(t, db, &sdk.Group{Name: sdk.RandomString(5)})
	localConsumer, err := authentication.LoadUserConsumerByTypeAndUserID(context.TODO(), api.mustDB(), sdk.ConsumerLocal, usr.ID, authentication.LoadUserConsumerOptions.WithAuthentifiedUser)
	require.NoError(t, err)

	consumerOptions := builtin.NewConsumerOptions{
		Name:        sdk.RandomString(10),
		Description: sdk.RandomString(10),
		Duration:    0,
		GroupIDs:    usr.GetGroupIDs(),
		Scopes:      sdk.NewAuthConsumerScopeDetails(sdk.AuthConsumerScopeProject),
	}
	_, jws, err := builtin.NewConsumer(context.TODO(), db, consumerOptions, localConsumer)
	require.NoError(t, err)
	AuthentififyBuiltinConsumer(t, api, jws, nil)
}
