package api

import (
	"context"
	"encoding/json"
	"github.com/ovh/cds/engine/api/link"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/sdk"
	"github.com/stretchr/testify/require"
	"net/http/httptest"
	"testing"
)

func Test_getUserLinksHandler(t *testing.T) {
	api, db, _ := newTestAPI(t)

	u, pass := assets.InsertAdminUser(t, db)

	ul := sdk.UserLink{
		AuthentifiedUserID: u.ID,
		Username:           sdk.RandomString(10),
		Type:               "github",
	}

	require.NoError(t, link.Insert(context.Background(), db, &ul))

	//Prepare request
	vars := map[string]string{
		"permUsername": u.Username,
	}
	uri := api.Router.GetRoute("GET", api.getUserLinksHandler, vars)
	test.NotEmpty(t, uri)
	req := assets.NewAuthentifiedRequest(t, u, pass, "GET", uri, nil)

	//Do the request
	rec := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	require.Equal(t, 200, rec.Code)

	var uls []sdk.UserLink
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &uls))

	require.Equal(t, 1, len(uls))
	require.Equal(t, ul.Username, uls[0].Username)

}
