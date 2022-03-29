package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/stretchr/testify/require"

	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/engine/api/user"
	"github.com/ovh/cds/sdk"
)

func Test_getUserContactsHandler(t *testing.T) {
	api, db, _ := newTestAPI(t)

	u, jwtRaw := assets.InsertLambdaUser(t, db)

	seed := sdk.RandomString(20)
	require.NoError(t, user.InsertContact(context.TODO(), db, &sdk.UserContact{
		Primary: true,
		Type:    sdk.UserContactTypeEmail,
		UserID:  u.ID,
		Value:   seed + "@lolcat.local",
	}))
	require.NoError(t, user.InsertContact(context.TODO(), db, &sdk.UserContact{
		Primary: false,
		Type:    sdk.UserContactTypeEmail,
		UserID:  u.ID,
		Value:   seed + "@lolcat2.host",
	}))

	uri := api.Router.GetRoute(http.MethodGet, api.getUserContactsHandler, map[string]string{
		"permUsername": u.Username,
	})
	require.NotEmpty(t, uri)
	req := assets.NewJWTAuthentifiedRequest(t, jwtRaw, http.MethodGet, uri, nil)
	rec := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	require.Equal(t, 200, rec.Code)

	var cs []sdk.UserContact
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &cs))
	require.Equal(t, 2, len(cs))
	assert.Equal(t, seed+"@lolcat.local", cs[0].Value)
	assert.Equal(t, seed+"@lolcat2.host", cs[1].Value)
}
