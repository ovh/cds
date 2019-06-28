package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ovh/cds/engine/api/authentication"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ovh/cds/engine/api/authentication/builtin"
	"github.com/ovh/cds/engine/api/authentication/local"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/sdk"
)

func Test_getConsumersByUserHandler(t *testing.T) {
	api, db, _, end := newTestAPI(t)
	defer end()

	u, jwtRaw := assets.InsertLambdaUser(db)

	consumer, err := local.NewConsumer(db, u.ID, sdk.RandomString(20))
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
	require.Equal(t, 1, len(cs))
	require.Equal(t, consumer.Name, cs[0].Name)
}

func Test_postConsumerByUserHandler(t *testing.T) {
	api, db, _, end := newTestAPI(t)
	defer end()

	g := assets.InsertGroup(t, db)
	u, jwtRaw := assets.InsertLambdaUser(db, g)

	data := sdk.AuthConsumer{
		Name:     sdk.RandomString(10),
		GroupIDs: []int64{g.ID},
		Scopes:   []sdk.AuthConsumerScope{sdk.AuthConsumerScopeAccessToken},
	}

	uri := api.Router.GetRoute(http.MethodPost, api.postConsumerByUserHandler, map[string]string{
		"permUsername": u.Username,
	})
	require.NotEmpty(t, uri)
	req := assets.NewJWTAuthentifiedRequest(t, jwtRaw, http.MethodPost, uri, data)
	rec := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	require.Equal(t, 201, rec.Code)

	var created sdk.AuthConsumerCreateResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &created))
	assert.NotEmpty(t, created.Token)
	assert.Equal(t, data.Name, created.Consumer.Name)
	require.Equal(t, 1, len(created.Consumer.GroupIDs))
	require.Equal(t, g.ID, created.Consumer.GroupIDs[0])
	require.Equal(t, 1, len(created.Consumer.Scopes))
	require.Equal(t, sdk.AuthConsumerScopeAccessToken, created.Consumer.Scopes[0])
}

func Test_deleteConsumerByUserHandler(t *testing.T) {
	api, db, _, end := newTestAPI(t)
	defer end()

	u, jwtRaw := assets.InsertLambdaUser(db)

	localConsumer, err := authentication.LoadConsumerByTypeAndUserID(context.TODO(), db, sdk.ConsumerLocal, u.ID)
	require.NoError(t, err)
	newConsumer, _, err := builtin.NewConsumer(db, sdk.RandomString(10), "", localConsumer, nil, []sdk.AuthConsumerScope{sdk.AuthConsumerScopeAccessToken})
	require.NoError(t, err)
	cs, err := authentication.LoadConsumersByUserID(context.TODO(), db, u.ID)
	require.NoError(t, err)
	assert.Equal(t, 2, len(cs))

	uri := api.Router.GetRoute(http.MethodDelete, api.deleteConsumerByUserHandler, map[string]string{
		"permUsername":   u.Username,
		"permConsumerID": newConsumer.ID,
	})
	require.NotEmpty(t, uri)
	req := assets.NewJWTAuthentifiedRequest(t, jwtRaw, http.MethodDelete, uri, nil)
	rec := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	require.Equal(t, 200, rec.Code)

	cs, err = authentication.LoadConsumersByUserID(context.TODO(), db, u.ID)
	require.NoError(t, err)
	assert.Equal(t, 1, len(cs))
}
