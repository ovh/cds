package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ovh/cds/engine/api/authentication"
	"github.com/ovh/cds/engine/api/authentication/builtin"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/sdk"
)

func Test_getConsumersByUserHandler(t *testing.T) {
	api, db, _, end := newTestAPI(t)
	defer end()

	u, jwtRaw := assets.InsertLambdaUser(t, db)
	localConsumer, err := authentication.LoadConsumerByTypeAndUserID(context.TODO(), db, sdk.ConsumerLocal, u.ID,
		authentication.LoadConsumerOptions.WithAuthentifiedUser)
	require.NoError(t, err)

	consumer, _, err := builtin.NewConsumer(context.TODO(), db, sdk.RandomString(10), "", localConsumer, nil, []sdk.AuthConsumerScope{sdk.AuthConsumerScopeUser})
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
	assert.Equal(t, localConsumer.ID, cs[0].ID)
	assert.Equal(t, consumer.ID, cs[1].ID)

	uri = api.Router.GetRoute(http.MethodGet, api.getConsumersByUserHandler, map[string]string{
		"permUsername": "me",
	})
	require.NotEmpty(t, uri)
	req = assets.NewJWTAuthentifiedRequest(t, jwtRaw, http.MethodGet, uri, nil)
	rec = httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	require.Equal(t, 200, rec.Code)

	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &cs))
	require.Equal(t, 2, len(cs))
	assert.Equal(t, localConsumer.ID, cs[0].ID)
	assert.Equal(t, consumer.ID, cs[1].ID)
}

func Test_postConsumerByUserHandler(t *testing.T) {
	api, db, _, end := newTestAPI(t)
	defer end()

	g := assets.InsertGroup(t, db)
	u, jwtRaw := assets.InsertLambdaUser(t, db, g)
	localConsumer, err := authentication.LoadConsumerByTypeAndUserID(context.TODO(), db, sdk.ConsumerLocal, u.ID)
	require.NoError(t, err)
	_, jwtRawAdmin := assets.InsertAdminUser(t, db)

	data := sdk.AuthConsumer{
		Name:     sdk.RandomString(10),
		GroupIDs: []int64{g.ID},
		Scopes:   []sdk.AuthConsumerScope{sdk.AuthConsumerScopeAccessToken},
		IssuedAt: time.Now(),
	}

	uri := api.Router.GetRoute(http.MethodPost, api.postConsumerByUserHandler, map[string]string{
		"permUsername": u.Username,
	})
	require.NotEmpty(t, uri)
	req := assets.NewJWTAuthentifiedRequest(t, jwtRawAdmin, http.MethodPost, uri, data)
	rec := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	require.Equal(t, 403, rec.Code)

	uri = api.Router.GetRoute(http.MethodPost, api.postConsumerByUserHandler, map[string]string{
		"permUsername": u.Username,
	})
	require.NotEmpty(t, uri)
	req = assets.NewJWTAuthentifiedRequest(t, jwtRaw, http.MethodPost, uri, data)
	rec = httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	require.Equal(t, 201, rec.Code)

	var created sdk.AuthConsumerCreateResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &created))
	assert.NotEmpty(t, created.Token)
	assert.Equal(t, data.Name, created.Consumer.Name)
	require.Equal(t, 1, len(created.Consumer.GroupIDs))
	assert.Equal(t, g.ID, created.Consumer.GroupIDs[0])
	require.Equal(t, 1, len(created.Consumer.Scopes))
	assert.Equal(t, sdk.AuthConsumerScopeAccessToken, created.Consumer.Scopes[0])
	assert.Equal(t, localConsumer.ID, *created.Consumer.ParentID)
}

func Test_deleteConsumerByUserHandler(t *testing.T) {
	api, db, _, end := newTestAPI(t)
	defer end()

	u, jwtRaw := assets.InsertLambdaUser(t, db)

	localConsumer, err := authentication.LoadConsumerByTypeAndUserID(context.TODO(), db, sdk.ConsumerLocal, u.ID,
		authentication.LoadConsumerOptions.WithAuthentifiedUser)
	require.NoError(t, err)
	newConsumer, _, err := builtin.NewConsumer(context.TODO(), db, sdk.RandomString(10), "", localConsumer, nil, []sdk.AuthConsumerScope{sdk.AuthConsumerScopeAccessToken})
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

func Test_postConsumerRegenByUserHandler(t *testing.T) {
	api, db, _, end := newTestAPI(t)
	defer end()

	u, jwtRaw := assets.InsertLambdaUser(t, db)
	localConsumer, err := authentication.LoadConsumerByTypeAndUserID(context.TODO(), db, sdk.ConsumerLocal, u.ID,
		authentication.LoadConsumerOptions.WithAuthentifiedUser)
	require.NoError(t, err)

	// Test that we can't regen a no builtin consumer
	uri := api.Router.GetRoute(http.MethodPost, api.postConsumerRegenByUserHandler, map[string]string{
		"permUsername":   u.Username,
		"permConsumerID": localConsumer.ID,
	})
	req := assets.NewJWTAuthentifiedRequest(t, jwtRaw, http.MethodPost, uri, sdk.AuthConsumerRegenRequest{})
	rec := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusForbidden, rec.Code)

	builtinConsumer, signinToken1, err := builtin.NewConsumer(context.TODO(), db, sdk.RandomString(10), "", localConsumer, nil, []sdk.AuthConsumerScope{sdk.AuthConsumerScopeUser, sdk.AuthConsumerScopeAccessToken})
	require.NoError(t, err)
	session, err := authentication.NewSession(context.TODO(), db, builtinConsumer, 5*time.Minute, false)
	require.NoError(t, err, "cannot create session")
	jwt2, err := authentication.NewSessionJWT(session)
	require.NoError(t, err, "cannot create jwt")

	uri = api.Router.GetRoute(http.MethodGet, api.getUserHandler, map[string]string{
		"permUsernamePublic": "me",
	})
	require.NotEmpty(t, uri)
	req = assets.NewJWTAuthentifiedRequest(t, jwtRaw, http.MethodGet, uri, nil)
	rec = httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	req = assets.NewJWTAuthentifiedRequest(t, jwt2, http.MethodGet, uri, nil)
	rec = httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	// Wait 2 seconds before regen
	time.Sleep(2 * time.Second)

	uri = api.Router.GetRoute(http.MethodPost, api.postConsumerRegenByUserHandler, map[string]string{
		"permUsername":   u.Username,
		"permConsumerID": builtinConsumer.ID,
	})
	req = assets.NewJWTAuthentifiedRequest(t, jwt2, http.MethodPost, uri, sdk.AuthConsumerRegenRequest{
		RevokeSessions: true,
	})
	rec = httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	var response sdk.AuthConsumerCreateResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))

	t.Logf("%+v", response)

	session, err = authentication.NewSession(context.TODO(), db, builtinConsumer, 5*time.Minute, false)
	require.NoError(t, err)
	jwt3, err := authentication.NewSessionJWT(session)
	require.NoError(t, err)

	uri = api.Router.GetRoute(http.MethodGet, api.getUserHandler, map[string]string{
		"permUsernamePublic": "me",
	})

	// The new token should be ok
	req = assets.NewJWTAuthentifiedRequest(t, jwt3, http.MethodGet, uri, nil)
	rec = httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	// After the regen the old token should be invalidated because we choose to drop the sessions
	req = assets.NewJWTAuthentifiedRequest(t, jwt2, http.MethodGet, uri, nil)
	rec = httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusUnauthorized, rec.Code)

	// the old signing token from the builtin consumer should be invalidated
	uri = api.Router.GetRoute(http.MethodPost, api.postAuthBuiltinSigninHandler, nil)
	req = assets.NewRequest(t, "POST", uri, sdk.AuthConsumerSigninRequest{
		"token": signinToken1,
	})
	rec = httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusBadRequest, rec.Code)

	// the new signing token from the builtin consumer should be fine
	uri = api.Router.GetRoute(http.MethodPost, api.postAuthBuiltinSigninHandler, nil)
	req = assets.NewRequest(t, "POST", uri, sdk.AuthConsumerSigninRequest{
		"token": response.Token,
	})
	rec = httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
}

func Test_getSessionsByUserHandler(t *testing.T) {
	api, db, _, end := newTestAPI(t)
	defer end()

	u, jwtRaw := assets.InsertLambdaUser(t, db)
	localConsumer, err := authentication.LoadConsumerByTypeAndUserID(context.TODO(), db, sdk.ConsumerLocal, u.ID,
		authentication.LoadConsumerOptions.WithAuthentifiedUser)
	require.NoError(t, err)

	consumer, _, err := builtin.NewConsumer(context.TODO(), db, sdk.RandomString(10), "", localConsumer, nil, []sdk.AuthConsumerScope{sdk.AuthConsumerScopeUser})
	require.NoError(t, err)
	s2, err := authentication.NewSession(context.TODO(), db, consumer, time.Second, false)
	require.NoError(t, err)
	s3, err := authentication.NewSession(context.TODO(), db, consumer, time.Second, false)
	require.NoError(t, err)

	uri := api.Router.GetRoute(http.MethodGet, api.getSessionsByUserHandler, map[string]string{
		"permUsername": u.Username,
	})
	require.NotEmpty(t, uri)
	req := assets.NewJWTAuthentifiedRequest(t, jwtRaw, http.MethodGet, uri, nil)
	rec := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	require.Equal(t, 200, rec.Code)

	var ss []sdk.AuthSession
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &ss))
	require.Equal(t, 3, len(ss))
	assert.Equal(t, localConsumer.ID, ss[0].ConsumerID)
	assert.Equal(t, s2.ID, ss[1].ID)
	assert.Equal(t, s3.ID, ss[2].ID)
}

func Test_deleteSessionByUserHandler(t *testing.T) {
	api, db, _, end := newTestAPI(t)
	defer end()

	u, jwtRaw := assets.InsertLambdaUser(t, db)
	localConsumer, err := authentication.LoadConsumerByTypeAndUserID(context.TODO(), db, sdk.ConsumerLocal, u.ID,
		authentication.LoadConsumerOptions.WithAuthentifiedUser)
	require.NoError(t, err)

	consumer, _, err := builtin.NewConsumer(context.TODO(), db, sdk.RandomString(10), "", localConsumer, nil, []sdk.AuthConsumerScope{sdk.AuthConsumerScopeUser})
	require.NoError(t, err)
	s2, err := authentication.NewSession(context.TODO(), db, consumer, time.Second, false)
	require.NoError(t, err)

	ss, err := authentication.LoadSessionsByConsumerIDs(context.TODO(), db, []string{localConsumer.ID, consumer.ID})
	require.NoError(t, err)
	assert.Equal(t, 2, len(ss))

	uri := api.Router.GetRoute(http.MethodDelete, api.deleteSessionByUserHandler, map[string]string{
		"permUsername":  u.Username,
		"permSessionID": s2.ID,
	})
	require.NotEmpty(t, uri)
	req := assets.NewJWTAuthentifiedRequest(t, jwtRaw, http.MethodDelete, uri, nil)
	rec := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	require.Equal(t, 200, rec.Code)

	ss, err = authentication.LoadSessionsByConsumerIDs(context.TODO(), db, []string{localConsumer.ID, consumer.ID})
	require.NoError(t, err)
	assert.Equal(t, 1, len(ss))
}
