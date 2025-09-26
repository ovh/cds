package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ovh/cds/engine/api/link"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/sdk"
	"github.com/stretchr/testify/require"
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

func Test_postUserLinkHandler(t *testing.T) {
	api, db, _ := newTestAPI(t)

	// Admin user (target user for link creation and auth user)
	adminUser, adminJWT := assets.InsertAdminUser(t, db)

	// Lambda user (non-admin) for forbidden scenario
	_, lambdaJWT := assets.InsertLambdaUser(t, db)

	// Base vars for route
	vars := map[string]string{"permUsername": adminUser.Username}
	uri := api.Router.GetRoute(http.MethodPost, api.postUserLinkHandler, vars)
	test.NotEmpty(t, uri)

	// 1. Happy path
	payload := sdk.UserLink{
		Type:       string(sdk.ConsumerBitbucketServer), // not a valid AuthConsumerType so passes the IsValid() check inversion
		ExternalID: sdk.RandomString(8),
		Username:   "ext-" + sdk.RandomString(6),
	}
	req := assets.NewJWTAuthentifiedRequest(t, adminJWT, http.MethodPost, uri, payload)
	rec := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusNoContent, rec.Code)

	// Check it is inserted
	links, err := link.LoadUserLinksByUserID(context.Background(), db, adminUser.ID)
	require.NoError(t, err)
	require.Len(t, links, 1)
	require.Equal(t, payload.Type, links[0].Type)
	require.Equal(t, payload.Username, links[0].Username)

	// 2. Conflict (same type again)
	conflictPayload := sdk.UserLink{
		Type:       payload.Type,
		ExternalID: sdk.RandomString(8),
		Username:   "other-" + sdk.RandomString(5),
	}
	req = assets.NewJWTAuthentifiedRequest(t, adminJWT, http.MethodPost, uri, conflictPayload)
	rec = httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusConflict, rec.Code)

	// 3. Invalid body (missing fields)
	invalidPayload := sdk.UserLink{Type: "foo"}
	req = assets.NewJWTAuthentifiedRequest(t, adminJWT, http.MethodPost, uri, invalidPayload)
	rec = httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusBadRequest, rec.Code)

	// 4. Forbidden (non admin tries to add link for admin target)
	req = assets.NewJWTAuthentifiedRequest(t, lambdaJWT, http.MethodPost, uri, sdk.UserLink{
		Type:       string(sdk.ConsumerBitbucketServer),
		ExternalID: sdk.RandomString(8),
		Username:   "u-" + sdk.RandomString(4),
	})
	rec = httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusForbidden, rec.Code)

	// 5. Invalid consumer
	req = assets.NewJWTAuthentifiedRequest(t, adminJWT, http.MethodPost, uri, sdk.UserLink{
		Type:       "fake",
		ExternalID: sdk.RandomString(8),
		Username:   "u-" + sdk.RandomString(4),
	})
	rec = httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusBadRequest, rec.Code)

	// 6. Delete link
	varDelete := map[string]string{"permUsername": adminUser.Username, "consumerType": string(sdk.ConsumerBitbucketServer)}
	uriDelete := api.Router.GetRoute(http.MethodDelete, api.deleteUserLinkHandler, varDelete)
	test.NotEmpty(t, uriDelete)

	reqDel := assets.NewJWTAuthentifiedRequest(t, adminJWT, http.MethodDelete, uriDelete, nil)
	recDel := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(recDel, reqDel)
	require.Equal(t, http.StatusNoContent, recDel.Code)

	links, err = link.LoadUserLinksByUserID(context.Background(), db, adminUser.ID)
	require.NoError(t, err)
	require.Len(t, links, 0)

}
