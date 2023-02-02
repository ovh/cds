package api

import (
	"encoding/json"
	"github.com/ovh/cds/engine/api/link"
	"github.com/ovh/cds/engine/api/link/github"
	"github.com/ovh/cds/sdk"
	"github.com/stretchr/testify/require"
	"net/http/httptest"
	"testing"

	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
)

func Test_postAskLinkExternalUserWithCDSHandler(t *testing.T) {
	api, db, _ := newTestAPI(t)
	api.LinkDrivers = map[sdk.AuthConsumerType]link.LinkDriver{
		"github": github.NewLinkGithubDriver(
			"http://lolcat.host",
			"https://github.com",
			"https://api.github.com",
			"clientID",
			"clientSecret"),
	}

	u, pass := assets.InsertAdminUser(t, db)

	//Prepare request
	vars := map[string]string{
		"consumerType": "github",
	}
	uri := api.Router.GetRoute("POST", api.postAskLinkExternalUserWithCDSHandler, vars)
	test.NotEmpty(t, uri)
	req := assets.NewAuthentifiedRequest(t, u, pass, "POST", uri, nil)

	//Do the request
	rec := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	require.Equal(t, 200, rec.Code)

	//Check result
	var result sdk.AuthDriverSigningRedirect
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &result))
	require.Equal(t, result.Method, "GET")
	require.Contains(t, result.URL, "redirect_uri=http://lolcat.host/auth/callback/github")
}
