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

func AuthentififyBuiltinConsumer(t *testing.T, api *API, jwsToken string) string {
	uri := api.Router.GetRoute("POST", api.postAuthBuiltinSigninHandler, nil)
	require.NotEmpty(t, uri)
	btes, err := json.Marshal(sdk.AuthConsumerSigninRequest{"token": jwsToken})
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
	localConsumer, err := authentication.LoadConsumerByTypeAndUserID(context.TODO(), api.mustDB(), sdk.ConsumerLocal, usr.ID, authentication.LoadConsumerOptions.WithAuthentifiedUser)
	require.NoError(t, err)

	_, jws, err := builtin.NewConsumer(context.TODO(), db, sdk.RandomString(10), sdk.RandomString(10), 0, localConsumer, usr.GetGroupIDs(),
		sdk.NewAuthConsumerScopeDetails(sdk.AuthConsumerScopeProject))
	require.NoError(t, err)
	AuthentififyBuiltinConsumer(t, api, jws)
}
