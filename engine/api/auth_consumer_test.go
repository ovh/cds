package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
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
	api, db, _ := newTestAPI(t)

	u, jwtRaw := assets.InsertLambdaUser(t, db)
	localConsumer, err := authentication.LoadUserConsumerByTypeAndUserID(context.TODO(), db, sdk.ConsumerLocal, u.ID,
		authentication.LoadUserConsumerOptions.WithAuthentifiedUser)
	require.NoError(t, err)

	consumerOptions := builtin.NewConsumerOptions{
		Name:        sdk.RandomString(10),
		Description: sdk.RandomString(10),
		Duration:    0,
		Scopes:      sdk.NewAuthConsumerScopeDetails(sdk.AuthConsumerScopeUser),
	}
	consumer, _, err := builtin.NewConsumer(context.TODO(), db, consumerOptions, localConsumer)
	require.NoError(t, err)

	uri := api.Router.GetRoute(http.MethodGet, api.getConsumersByUserHandler, map[string]string{
		"permUsername": u.Username,
	})
	require.NotEmpty(t, uri)
	req := assets.NewJWTAuthentifiedRequest(t, jwtRaw, http.MethodGet, uri, nil)
	rec := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	require.Equal(t, 200, rec.Code)

	var cs []sdk.AuthUserConsumer
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
	require.Len(t, cs, 2)
	require.Equal(t, localConsumer.ID, cs[0].ID)
	require.True(t, slices.ContainsFunc(cs, func(c sdk.AuthUserConsumer) bool { return c.ID == localConsumer.ID }))
	require.True(t, slices.ContainsFunc(cs, func(c sdk.AuthUserConsumer) bool { return c.ID == consumer.ID }))
}

func Test_postConsumerByUserHandler(t *testing.T) {
	api, db, _ := newTestAPI(t)

	g := assets.InsertGroup(t, db)
	u, jwtRaw := assets.InsertLambdaUser(t, db, g)
	localConsumer, err := authentication.LoadUserConsumerByTypeAndUserID(context.TODO(), db, sdk.ConsumerLocal, u.ID)
	require.NoError(t, err)
	_, jwtRawAdmin := assets.InsertAdminUser(t, db)

	data := sdk.AuthUserConsumer{
		AuthConsumer: sdk.AuthConsumer{
			Name:            sdk.RandomString(10),
			ValidityPeriods: sdk.NewAuthConsumerValidityPeriod(time.Now(), 0),
		},
		AuthConsumerUser: sdk.AuthUserConsumerData{
			GroupIDs:     []int64{g.ID},
			ScopeDetails: sdk.NewAuthConsumerScopeDetails(sdk.AuthConsumerScopeAccessToken),
		},
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
	require.NotEmpty(t, created.Token)
	require.Equal(t, data.Name, created.Consumer.Name)
	require.Len(t, created.Consumer.AuthConsumerUser.GroupIDs, 1)
	require.Equal(t, g.ID, created.Consumer.AuthConsumerUser.GroupIDs[0])
	require.Len(t, created.Consumer.AuthConsumerUser.ScopeDetails, 1)
	require.Equal(t, sdk.AuthConsumerScopeAccessToken, created.Consumer.AuthConsumerUser.ScopeDetails[0].Scope)
	require.Equal(t, localConsumer.ID, *created.Consumer.ParentID)
}

func Test_deleteConsumerByUserHandler(t *testing.T) {
	api, db, _ := newTestAPI(t)

	u, jwtRaw := assets.InsertLambdaUser(t, db)

	localConsumer, err := authentication.LoadUserConsumerByTypeAndUserID(context.TODO(), db, sdk.ConsumerLocal, u.ID,
		authentication.LoadUserConsumerOptions.WithAuthentifiedUser)
	require.NoError(t, err)
	consumerOptions := builtin.NewConsumerOptions{
		Name:        sdk.RandomString(10),
		Description: sdk.RandomString(10),
		Duration:    0,
		Scopes:      sdk.NewAuthConsumerScopeDetails(sdk.AuthConsumerScopeAccessToken),
	}
	newConsumer, _, err := builtin.NewConsumer(context.TODO(), db, consumerOptions, localConsumer)
	require.NoError(t, err)
	cs, err := authentication.LoadUserConsumersByUserID(context.TODO(), db, u.ID)
	require.NoError(t, err)
	require.Len(t, cs, 2)

	uri := api.Router.GetRoute(http.MethodDelete, api.deleteConsumerByUserHandler, map[string]string{
		"permUsername":   u.Username,
		"permConsumerID": newConsumer.ID,
	})
	require.NotEmpty(t, uri)
	req := assets.NewJWTAuthentifiedRequest(t, jwtRaw, http.MethodDelete, uri, nil)
	rec := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	require.Equal(t, 200, rec.Code)

	cs, err = authentication.LoadUserConsumersByUserID(context.TODO(), db, u.ID)
	require.NoError(t, err)
	require.Len(t, cs, 1)
}

func Test_postConsumerRegenByUserHandler(t *testing.T) {
	api, db, _ := newTestAPI(t)

	u, jwtRaw := assets.InsertLambdaUser(t, db)
	localConsumer, err := authentication.LoadUserConsumerByTypeAndUserID(context.TODO(), db, sdk.ConsumerLocal, u.ID,
		authentication.LoadUserConsumerOptions.WithAuthentifiedUser)
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

	consumerOptions := builtin.NewConsumerOptions{
		Name:        sdk.RandomString(10),
		Description: sdk.RandomString(10),
		Duration:    0,
		Scopes:      sdk.NewAuthConsumerScopeDetails(sdk.AuthConsumerScopeUser, sdk.AuthConsumerScopeAccessToken),
	}

	builtinConsumer, signinToken1, err := builtin.NewConsumer(context.TODO(), db, consumerOptions, localConsumer)
	require.NoError(t, err)
	session, err := authentication.NewSession(context.TODO(), db, &builtinConsumer.AuthConsumer, 5*time.Minute)
	require.NoError(t, err, "cannot create session")
	jwt2, err := authentication.NewSessionJWT(session, "")
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

	session, err = authentication.NewSession(context.TODO(), db, &builtinConsumer.AuthConsumer, 5*time.Minute)
	require.NoError(t, err)
	jwt3, err := authentication.NewSessionJWT(session, "")
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
	require.Equal(t, http.StatusForbidden, rec.Code)

	// the new signing token from the builtin consumer should be fine
	signinToken2 := response.Token
	uri = api.Router.GetRoute(http.MethodPost, api.postAuthBuiltinSigninHandler, nil)
	req = assets.NewRequest(t, "POST", uri, sdk.AuthConsumerSigninRequest{
		"token": signinToken2,
	})
	rec = httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	t.Log("next use case...")
	time.Sleep(2 * time.Second)

	// Regen the latest token with an overlap duration
	uri = api.Router.GetRoute(http.MethodPost, api.postConsumerRegenByUserHandler, map[string]string{
		"permUsername":   u.Username,
		"permConsumerID": builtinConsumer.ID,
	})
	req = assets.NewJWTAuthentifiedRequest(t, jwt3, http.MethodPost, uri, sdk.AuthConsumerRegenRequest{
		RevokeSessions:  true,
		OverlapDuration: "4s", // short 4s overlap
		NewDuration:     1,
	})
	rec = httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
	t.Logf("here is the new consumer: %+v", response)

	signinToken3 := response.Token

	// Wait before using it
	time.Sleep(2 * time.Second)

	// the new signing token from the builtin consumer should be fine
	uri = api.Router.GetRoute(http.MethodPost, api.postAuthBuiltinSigninHandler, nil)
	req = assets.NewRequest(t, "POST", uri, sdk.AuthConsumerSigninRequest{
		"token": signinToken3,
	})
	rec = httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	// The old token should be ok too, because of the overlap duration
	uri = api.Router.GetRoute(http.MethodPost, api.postAuthBuiltinSigninHandler, nil)
	req = assets.NewRequest(t, "POST", uri, sdk.AuthConsumerSigninRequest{
		"token": signinToken2,
	})
	rec = httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	// Now wait for the overlab duration to be over....
	time.Sleep(2 * time.Second)

	// Now, the old token should be rejected
	uri = api.Router.GetRoute(http.MethodPost, api.postAuthBuiltinSigninHandler, nil)
	req = assets.NewRequest(t, "POST", uri, sdk.AuthConsumerSigninRequest{
		"token": signinToken2,
	})
	rec = httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusForbidden, rec.Code)
}

func Test_getSessionsByUserHandler(t *testing.T) {
	api, db, _ := newTestAPI(t)

	u, jwtRaw := assets.InsertLambdaUser(t, db)
	localConsumer, err := authentication.LoadUserConsumerByTypeAndUserID(context.TODO(), db, sdk.ConsumerLocal, u.ID,
		authentication.LoadUserConsumerOptions.WithAuthentifiedUser)
	require.NoError(t, err)

	consumerOptions := builtin.NewConsumerOptions{
		Name:   sdk.RandomString(10),
		Scopes: sdk.NewAuthConsumerScopeDetails(sdk.AuthConsumerScopeUser),
	}
	consumer, _, err := builtin.NewConsumer(context.TODO(), db, consumerOptions, localConsumer)
	require.NoError(t, err)
	s2, err := authentication.NewSession(context.TODO(), db, &consumer.AuthConsumer, time.Second)
	require.NoError(t, err)
	s3, err := authentication.NewSession(context.TODO(), db, &consumer.AuthConsumer, time.Second)
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
	require.Len(t, ss, 3)
	require.True(t, slices.ContainsFunc(ss, func(e sdk.AuthSession) bool { return e.ConsumerID == localConsumer.ID }))
	require.True(t, slices.ContainsFunc(ss, func(e sdk.AuthSession) bool { return e.ID == s2.ID }))
	require.True(t, slices.ContainsFunc(ss, func(e sdk.AuthSession) bool { return e.ID == s3.ID }))
}

func Test_deleteSessionByUserHandler(t *testing.T) {
	api, db, _ := newTestAPI(t)

	u, jwtRaw := assets.InsertLambdaUser(t, db)
	localConsumer, err := authentication.LoadUserConsumerByTypeAndUserID(context.TODO(), db, sdk.ConsumerLocal, u.ID,
		authentication.LoadUserConsumerOptions.WithAuthentifiedUser)
	require.NoError(t, err)

	consumerOptions := builtin.NewConsumerOptions{
		Name:   sdk.RandomString(10),
		Scopes: sdk.NewAuthConsumerScopeDetails(sdk.AuthConsumerScopeUser),
	}
	consumer, _, err := builtin.NewConsumer(context.TODO(), db, consumerOptions, localConsumer)
	require.NoError(t, err)
	s2, err := authentication.NewSession(context.TODO(), db, &consumer.AuthConsumer, time.Second)
	require.NoError(t, err)

	ss, err := authentication.LoadSessionsByConsumerIDs(context.TODO(), db, []string{localConsumer.ID, consumer.ID})
	require.NoError(t, err)
	require.Len(t, ss, 2)

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
	require.Len(t, ss, 1)
}
