package api

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/ovh/cds/engine/api/accesstoken"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/stretchr/testify/assert"

	"github.com/ovh/cds/sdk"
)

func TestAPI_TokenHandlers(t *testing.T) {
	test.NoError(t, accesstoken.Init("cds_test", test.TestKey))

	api, db, router, end := newTestAPI(t)
	defer end()

	grp := sdk.Group{Name: sdk.RandomString(10)}
	user, password := assets.InsertLambdaUser(db, &grp)
	test.NoError(t, group.SetUserGroupAdmin(db, grp.ID, user.ID))

	jwt, err := assets.NewJWTToken(t, db, *user, grp)
	test.NoError(t, err)

	uri := router.GetRoute("POST", api.postNewAccessTokenHandler, nil)
	req := assets.NewJWTAuthentifiedRequest(t, jwt, "POST", uri,
		sdk.AccessTokenRequest{
			Origin:                "test",
			Description:           "test",
			ExpirationDelaySecond: 3600,
			GroupsIDs:             []int64{grp.ID},
		},
	)

	// Do the request
	w := httptest.NewRecorder()

	router.Mux.ServeHTTP(w, req)
	assert.Equal(t, 201, w.Code)

	jwtToken := w.Header().Get("X-CDS-JWT")
	t.Logf("jwt token is %v...", jwtToken[:12])

	var accessToken sdk.AccessToken
	test.NoError(t, json.Unmarshal(w.Body.Bytes(), &accessToken))

	vars := map[string]string{
		"id": accessToken.ID,
	}
	uri = router.GetRoute("PUT", api.putRegenAccessTokenHandler, vars)
	req = assets.NewAuthentifiedRequest(t, user, password, "PUT", uri,
		sdk.AccessTokenRequest{
			Origin:                "test",
			Description:           "test",
			ExpirationDelaySecond: 3600,
			GroupsIDs:             []int64{grp.ID},
		},
	)

	// Do the request
	w = httptest.NewRecorder()

	router.Mux.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)

	jwtToken = w.Header().Get("X-CDS-JWT")
	t.Logf("jwt token is %v...", jwtToken[:12])
	t.Logf("access token is : %s", w.Body.String())
}
