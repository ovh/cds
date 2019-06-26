package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ovh/cds/engine/api/authentication"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/sdk"
)

func Test_getConsumersByUserHandler(t *testing.T) {
	api, db, _, end := newTestAPI(t)
	defer end()

	u, jwtRaw := assets.InsertLambdaUser(db)

	consumer, err := authentication.NewConsumerBuiltin(db, sdk.RandomString(10), "", u.ID,
		[]int64{}, []string{sdk.AccessTokenScopeALL})
	require.NoError(t, err)

	uri := api.Router.GetRoute(http.MethodGet, api.getConsumersByUserHandler, map[string]string{
		"permUsername": u.Username,
	})
	require.NotEmpty(t, uri)
	req := assets.NewJWTAuthentifiedRequest(t, jwtRaw, http.MethodGet, uri, nil)
	rec := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	require.Equal(t, 200, rec.Code)

	var cs []sdk.AuthConsumer
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &cs))
	require.Equal(t, 2, len(cs))
	require.Equal(t, consumer.Name, cs[1].Name)
}
