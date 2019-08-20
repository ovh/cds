package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
)

func Test_authMiddleware_WithAuth(t *testing.T) {
	api, db, _, end := newTestAPI(t)
	defer end()

	u, jwt := assets.InsertLambdaUser(t, db)

	config := &service.HandlerConfig{}
	Auth(true)(config)

	req := assets.NewRequest(t, http.MethodGet, "", nil)
	w := httptest.NewRecorder()
	ctx, err := api.authMiddleware(context.TODO(), w, req, config)
	assert.Error(t, err, "an error should be returned because no jwt was given and auth is required")

	req = assets.NewJWTAuthentifiedRequest(t, jwt, http.MethodGet, "", nil)
	w = httptest.NewRecorder()
	ctx, err = api.authMiddleware(context.TODO(), w, req, config)
	assert.NoError(t, err, "no error should be returned because a jwt was given and is valid")
	require.NotNil(t, getAPIConsumer(ctx))
	assert.Equal(t, u.ID, getAPIConsumer(ctx).AuthentifiedUserID)

	req = assets.NewJWTAuthentifiedRequest(t, sdk.RandomString(10), http.MethodGet, "", nil)
	w = httptest.NewRecorder()
	ctx, err = api.authMiddleware(context.TODO(), w, req, config)
	assert.Error(t, err, "an error should be returned because a jwt was given but no valid session matching")
}

func Test_authMiddleware_WithoutAuth(t *testing.T) {
	api, db, _, end := newTestAPI(t)
	defer end()

	u, jwt := assets.InsertLambdaUser(t, db)

	config := &service.HandlerConfig{}
	Auth(false)(config)

	req := assets.NewRequest(t, http.MethodGet, "", nil)
	w := httptest.NewRecorder()
	ctx, err := api.authMiddleware(context.TODO(), w, req, config)
	assert.NoError(t, err, "no error should be returned because no jwt was given and auth not required")
	assert.Nil(t, getAPIConsumer(ctx))

	req = assets.NewJWTAuthentifiedRequest(t, jwt, http.MethodGet, "", nil)
	w = httptest.NewRecorder()
	ctx, err = api.authMiddleware(context.TODO(), w, req, config)
	assert.NoError(t, err, "no error should be returned because a jwt was given and is valid")
	require.NotNil(t, getAPIConsumer(ctx))
	assert.Equal(t, u.ID, getAPIConsumer(ctx).AuthentifiedUserID)

	req = assets.NewJWTAuthentifiedRequest(t, sdk.RandomString(10), http.MethodGet, "", nil)
	w = httptest.NewRecorder()
	ctx, err = api.authMiddleware(context.TODO(), w, req, config)
	assert.NoError(t, err, "no error should be returned for an invalid jwt when auth is not required")
	assert.Nil(t, getAPIConsumer(ctx))
}

func Test_authMiddleware_NeedAdmin(t *testing.T) {
	api, db, _, end := newTestAPI(t)
	defer end()

	_, jwtLambda := assets.InsertLambdaUser(t, db)
	admin, jwtAdmin := assets.InsertAdminUser(t, db)

	config := &service.HandlerConfig{}
	NeedAdmin(true)(config)

	req := assets.NewRequest(t, http.MethodGet, "", nil)
	w := httptest.NewRecorder()
	ctx, err := api.authMiddleware(context.TODO(), w, req, config)
	assert.Error(t, err, "an error should be returned because no jwt was given and admin auth is required")

	req = assets.NewJWTAuthentifiedRequest(t, jwtLambda, http.MethodGet, "", nil)
	w = httptest.NewRecorder()
	ctx, err = api.authMiddleware(context.TODO(), w, req, config)
	assert.Error(t, err, "an error should be returned because a jwt was given for a lambda user")

	req = assets.NewJWTAuthentifiedRequest(t, jwtAdmin, http.MethodGet, "", nil)
	w = httptest.NewRecorder()
	ctx, err = api.authMiddleware(context.TODO(), w, req, config)
	assert.NoError(t, err, "no error should be returned because a jwt was given for an admin user")
	require.NotNil(t, getAPIConsumer(ctx))
	assert.Equal(t, admin.ID, getAPIConsumer(ctx).AuthentifiedUserID)
}
