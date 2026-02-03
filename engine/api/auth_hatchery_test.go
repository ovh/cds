package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/sdk"
	"github.com/stretchr/testify/require"
)

func AuthentififyHatcheryConsumer(t *testing.T, api *API, jwsToken string, hatcheryName string) string {
	uri := api.Router.GetRouteV2(http.MethodPost, api.postAuthHatcherySigninHandler, nil)
	require.NotEmpty(t, uri)
	req := assets.NewRequest(t, http.MethodPost, uri, sdk.AuthConsumerHatcherySigninRequest{
		Name:  hatcheryName,
		Token: jwsToken,
	})
	rec := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	require.Equal(t, 200, rec.Code)

	var signinReponse sdk.AuthConsumerHatcherySigninResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &signinReponse))
	require.NotEmpty(t, signinReponse.Token, "session token should not be empty")

	return signinReponse.Token
}
