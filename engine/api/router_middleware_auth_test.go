package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ovh/cds/engine/api/authentication"
	"github.com/ovh/cds/engine/api/authentication/builtin"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
)

func Test_authMiddleware(t *testing.T) {
	api, db, _ := newTestAPI(t)

	u, jwt := assets.InsertLambdaUser(t, db)

	config := &service.HandlerConfig{}

	req := assets.NewRequest(t, http.MethodGet, "", nil)
	w := httptest.NewRecorder()
	ctx, err := api.jwtMiddleware(context.TODO(), w, req, config)
	require.NoError(t, err)
	ctx, err = api.authMiddleware(ctx, w, req, config)
	require.Error(t, err, "an error should be returned because no jwt was given and auth is required")

	req = assets.NewJWTAuthentifiedRequest(t, jwt, http.MethodGet, "", nil)
	w = httptest.NewRecorder()
	ctx, err = api.jwtMiddleware(context.TODO(), w, req, config)
	require.NoError(t, err)
	ctx, err = api.authMiddleware(ctx, w, req, config)
	require.NoError(t, err, "no error should be returned because a jwt was given and is valid")
	require.NotNil(t, getAPIConsumer(ctx))
	require.Equal(t, u.ID, getAPIConsumer(ctx).AuthentifiedUserID)

	req = assets.NewJWTAuthentifiedRequest(t, sdk.RandomString(10), http.MethodGet, "", nil)
	w = httptest.NewRecorder()
	ctx, err = api.jwtMiddleware(context.TODO(), w, req, config)
	require.NoError(t, err)
	ctx, err = api.authMiddleware(ctx, w, req, config)
	require.Error(t, err, "an error should be returned because a jwt was given but no valid session matching")
}

func Test_authMiddleware_WithAuthConsumerDisabled(t *testing.T) {
	api, db, _ := newTestAPI(t)

	g := assets.InsertGroup(t, db)
	u, _ := assets.InsertLambdaUser(t, db, g)
	localConsumer, err := authentication.LoadConsumerByTypeAndUserID(context.TODO(), db, sdk.ConsumerLocal, u.ID, authentication.LoadConsumerOptions.WithAuthentifiedUser)
	require.NoError(t, err)

	builtinConsumer, _, err := builtin.NewConsumer(context.TODO(), db, "builtin", "", localConsumer, []int64{g.ID},
		sdk.NewAuthConsumerScopeDetails(sdk.AuthConsumerScopes...))
	require.NoError(t, err)
	builtinSession, err := authentication.NewSession(context.TODO(), db, builtinConsumer, time.Second*5, false)
	require.NoError(t, err)
	jwt, err := authentication.NewSessionJWT(builtinSession)
	require.NoError(t, err)

	config := &service.HandlerConfig{}

	req := assets.NewJWTAuthentifiedRequest(t, jwt, http.MethodGet, "", nil)
	w := httptest.NewRecorder()
	ctx, err := api.jwtMiddleware(context.TODO(), w, req, config)
	require.NoError(t, err)
	_, err = api.authMiddleware(ctx, w, req, config)
	assert.NoError(t, err, "no error should be returned because a valid jwt was given")

	require.NoError(t, authentication.ConsumerRemoveGroup(context.TODO(), db, g))

	req = assets.NewJWTAuthentifiedRequest(t, jwt, http.MethodGet, "", nil)
	w = httptest.NewRecorder()
	ctx, err = api.jwtMiddleware(context.TODO(), w, req, config)
	require.NoError(t, err)
	_, err = api.authMiddleware(ctx, w, req, config)
	assert.Error(t, err, "an error should be returned because the consumer should have been disabled")
}

func Test_authOptionalMiddleware(t *testing.T) {
	api, db, _ := newTestAPI(t)

	u, jwt := assets.InsertLambdaUser(t, db)

	config := &service.HandlerConfig{}

	req := assets.NewRequest(t, http.MethodGet, "", nil)
	w := httptest.NewRecorder()
	ctx, err := api.jwtMiddleware(context.TODO(), w, req, config)
	require.NoError(t, err)
	ctx, err = api.authOptionalMiddleware(ctx, w, req, config)
	assert.NoError(t, err, "no error should be returned because no jwt was given and auth not required")
	assert.Nil(t, getAPIConsumer(ctx))

	req = assets.NewJWTAuthentifiedRequest(t, jwt, http.MethodGet, "", nil)
	w = httptest.NewRecorder()
	ctx, err = api.jwtMiddleware(context.TODO(), w, req, config)
	require.NoError(t, err)
	ctx, err = api.authOptionalMiddleware(ctx, w, req, config)
	assert.NoError(t, err, "no error should be returned because a jwt was given and is valid")
	require.NotNil(t, getAPIConsumer(ctx))
	assert.Equal(t, u.ID, getAPIConsumer(ctx).AuthentifiedUserID)

	req = assets.NewJWTAuthentifiedRequest(t, sdk.RandomString(10), http.MethodGet, "", nil)
	w = httptest.NewRecorder()
	ctx, err = api.jwtMiddleware(context.TODO(), w, req, config)
	require.NoError(t, err)
	ctx, err = api.authOptionalMiddleware(ctx, w, req, config)
	assert.NoError(t, err, "no error should be returned for an invalid jwt when auth is not required")
	assert.Nil(t, getAPIConsumer(ctx))
}

func Test_authAdminMiddleware(t *testing.T) {
	api, db, _ := newTestAPI(t)

	_, jwtLambda := assets.InsertLambdaUser(t, db)
	admin, jwtAdmin := assets.InsertAdminUser(t, db)

	config := &service.HandlerConfig{}

	req := assets.NewRequest(t, http.MethodGet, "", nil)
	w := httptest.NewRecorder()
	ctx, err := api.jwtMiddleware(context.TODO(), w, req, config)
	require.NoError(t, err)
	ctx, err = api.authAdminMiddleware(ctx, w, req, config)
	assert.Error(t, err, "an error should be returned because no jwt was given and admin auth is required")

	req = assets.NewJWTAuthentifiedRequest(t, jwtLambda, http.MethodGet, "", nil)
	w = httptest.NewRecorder()
	ctx, err = api.jwtMiddleware(context.TODO(), w, req, config)
	require.NoError(t, err)
	ctx, err = api.authAdminMiddleware(ctx, w, req, config)
	assert.Error(t, err, "an error should be returned because a jwt was given for a lambda user")

	req = assets.NewJWTAuthentifiedRequest(t, jwtAdmin, http.MethodGet, "", nil)
	w = httptest.NewRecorder()
	ctx, err = api.jwtMiddleware(context.TODO(), w, req, config)
	require.NoError(t, err)
	ctx, err = api.authAdminMiddleware(ctx, w, req, config)
	assert.NoError(t, err, "no error should be returned because a jwt was given for an admin user")
	require.NotNil(t, getAPIConsumer(ctx))
	assert.Equal(t, admin.ID, getAPIConsumer(ctx).AuthentifiedUserID)
}

func Test_authMiddleware_WithAuthConsumerScoped(t *testing.T) {
	api, db, _ := newTestAPI(t)

	g := assets.InsertGroup(t, db)
	u, _ := assets.InsertLambdaUser(t, db, g)
	localConsumer, err := authentication.LoadConsumerByTypeAndUserID(context.TODO(), db, sdk.ConsumerLocal, u.ID, authentication.LoadConsumerOptions.WithAuthentifiedUser)
	require.NoError(t, err)

	builtinConsumer, _, err := builtin.NewConsumer(context.TODO(), db, "builtin", "", localConsumer, []int64{g.ID}, []sdk.AuthConsumerScopeDetail{
		{
			Scope: sdk.AuthConsumerScopeAction,
			Endpoints: sdk.AuthConsumerScopeEndpoints{
				{
					Route:   "/my-handler2",
					Methods: []string{http.MethodGet},
				},
				{
					Route: "/my-handler3",
				},
			},
		},
		{
			Scope: sdk.AuthConsumerScopeAdmin,
		},
	})
	require.NoError(t, err)
	builtinSession, err := authentication.NewSession(context.TODO(), db, builtinConsumer, time.Second*5, false)
	require.NoError(t, err)
	jwt, err := authentication.NewSessionJWT(builtinSession)
	require.NoError(t, err)

	// GET /my-handler1 is forbidden (scope AccessToken required)
	configHandler1 := &service.HandlerConfig{
		CleanURL:      "/my-handler1",
		AllowedScopes: []sdk.AuthConsumerScope{sdk.AuthConsumerScopeAccessToken},
		Method:        http.MethodGet,
	}
	req := assets.NewJWTAuthentifiedRequest(t, jwt, configHandler1.Method, configHandler1.CleanURL, nil)
	w := httptest.NewRecorder()
	ctx, err := api.jwtMiddleware(context.TODO(), w, req, configHandler1)
	require.NoError(t, err)
	_, err = api.authMiddleware(ctx, w, req, configHandler1)
	assert.Error(t, err, "an error should be returned because consumer can't do GET on /my-handler1 missing scope")

	// GET /my-handler2 is authorized
	configHandler2 := &service.HandlerConfig{
		CleanURL:      "/my-handler2",
		AllowedScopes: []sdk.AuthConsumerScope{sdk.AuthConsumerScopeAction},
		Method:        http.MethodGet,
	}
	req = assets.NewJWTAuthentifiedRequest(t, jwt, configHandler2.Method, configHandler2.CleanURL, nil)
	w = httptest.NewRecorder()
	ctx, err = api.jwtMiddleware(context.TODO(), w, req, configHandler2)
	require.NoError(t, err)
	_, err = api.authMiddleware(ctx, w, req, configHandler2)
	assert.NoError(t, err, "no error should be returned because consumer can do GET on /my-handler2")

	// POST /my-handler2 is forbidden (missing POST method in scope Action)
	configHandler3 := &service.HandlerConfig{
		CleanURL:      "/my-handler2",
		AllowedScopes: []sdk.AuthConsumerScope{sdk.AuthConsumerScopeAction},
		Method:        http.MethodPost,
	}
	req = assets.NewJWTAuthentifiedRequest(t, jwt, configHandler3.Method, configHandler3.CleanURL, nil)
	w = httptest.NewRecorder()
	ctx, err = api.jwtMiddleware(context.TODO(), w, req, configHandler3)
	require.NoError(t, err)
	_, err = api.authMiddleware(ctx, w, req, configHandler3)
	assert.Error(t, err, "an error should be returned because consumer can't do POST on /my-handler2")

	// DELETE /my-handler3 is authorized as no method restriction set on route.
	configHandler4 := &service.HandlerConfig{
		CleanURL:      "/my-handler3",
		AllowedScopes: []sdk.AuthConsumerScope{sdk.AuthConsumerScopeAction},
		Method:        http.MethodDelete,
	}
	req = assets.NewJWTAuthentifiedRequest(t, jwt, configHandler4.Method, configHandler4.CleanURL, nil)
	w = httptest.NewRecorder()
	ctx, err = api.jwtMiddleware(context.TODO(), w, req, configHandler4)
	require.NoError(t, err)
	_, err = api.authMiddleware(ctx, w, req, configHandler4)
	assert.NoError(t, err, "no error should be returned because consumer can do any methods on /my-handler3")

	// PUT /my-handler4 is authorized as no route restriction set on scope.
	configHandler5 := &service.HandlerConfig{
		CleanURL:      "/my-handler4",
		AllowedScopes: []sdk.AuthConsumerScope{sdk.AuthConsumerScopeAdmin},
		Method:        http.MethodPut,
	}
	req = assets.NewJWTAuthentifiedRequest(t, jwt, configHandler5.Method, configHandler5.CleanURL, nil)
	w = httptest.NewRecorder()
	ctx, err = api.jwtMiddleware(context.TODO(), w, req, configHandler5)
	require.NoError(t, err)
	_, err = api.authMiddleware(ctx, w, req, configHandler5)
	assert.NoError(t, err, "no error should be returned because consumer can access any routes for scope Admin")
}
