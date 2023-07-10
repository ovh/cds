package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ovh/cds/engine/api/authentication"
	"github.com/ovh/cds/engine/api/authentication/builtin"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/hatchery"
	"github.com/ovh/cds/sdk/jws"
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
	require.NotNil(t, getUserConsumer(ctx))
	require.Equal(t, u.ID, getUserConsumer(ctx).AuthConsumerUser.AuthentifiedUserID)

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
	localConsumer, err := authentication.LoadUserConsumerByTypeAndUserID(context.TODO(), db, sdk.ConsumerLocal, u.ID, authentication.LoadUserConsumerOptions.WithAuthentifiedUser)
	require.NoError(t, err)

	consumerOptions := builtin.NewConsumerOptions{
		Name:     "builtin",
		GroupIDs: []int64{g.ID},
		Scopes:   sdk.NewAuthConsumerScopeDetails(sdk.AuthConsumerScopes...),
	}
	builtinConsumer, _, err := builtin.NewConsumer(context.TODO(), db, consumerOptions, localConsumer)
	require.NoError(t, err)
	builtinSession, err := authentication.NewSession(context.TODO(), db, &builtinConsumer.AuthConsumer, time.Second*5)
	require.NoError(t, err)
	jwt, err := authentication.NewSessionJWT(builtinSession, "")
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
	assert.Nil(t, getUserConsumer(ctx))

	req = assets.NewJWTAuthentifiedRequest(t, jwt, http.MethodGet, "", nil)
	w = httptest.NewRecorder()
	ctx, err = api.jwtMiddleware(context.TODO(), w, req, config)
	require.NoError(t, err)
	ctx, err = api.authOptionalMiddleware(ctx, w, req, config)
	assert.NoError(t, err, "no error should be returned because a jwt was given and is valid")
	require.NotNil(t, getUserConsumer(ctx))
	assert.Equal(t, u.ID, getUserConsumer(ctx).AuthConsumerUser.AuthentifiedUserID)

	req = assets.NewJWTAuthentifiedRequest(t, sdk.RandomString(10), http.MethodGet, "", nil)
	w = httptest.NewRecorder()
	ctx, err = api.jwtMiddleware(context.TODO(), w, req, config)
	require.NoError(t, err)
	ctx, err = api.authOptionalMiddleware(ctx, w, req, config)
	assert.NoError(t, err, "no error should be returned for an invalid jwt when auth is not required")
	assert.Nil(t, getUserConsumer(ctx))
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
	require.NotNil(t, getUserConsumer(ctx))
	assert.Equal(t, admin.ID, getUserConsumer(ctx).AuthConsumerUser.AuthentifiedUserID)
}

func Test_authMaintainerMiddleware(t *testing.T) {
	api, db, _ := newTestAPI(t)

	_, jwtLambda := assets.InsertLambdaUser(t, db)
	maintainer, jwtMaintainer := assets.InsertMaintainerUser(t, db)
	admin, jwtAdmin := assets.InsertAdminUser(t, db)

	config := &service.HandlerConfig{}

	req := assets.NewRequest(t, http.MethodGet, "", nil)
	w := httptest.NewRecorder()
	ctx, err := api.jwtMiddleware(context.TODO(), w, req, config)
	require.NoError(t, err)
	ctx, err = api.authMaintainerMiddleware(ctx, w, req, config)
	assert.Error(t, err, "an error should be returned because no jwt was given and maintainer auth is required")

	req = assets.NewJWTAuthentifiedRequest(t, jwtLambda, http.MethodGet, "", nil)
	w = httptest.NewRecorder()
	ctx, err = api.jwtMiddleware(context.TODO(), w, req, config)
	require.NoError(t, err)
	ctx, err = api.authMaintainerMiddleware(ctx, w, req, config)
	assert.Error(t, err, "an error should be returned because a jwt was given for a lambda user")

	req = assets.NewJWTAuthentifiedRequest(t, jwtMaintainer, http.MethodGet, "", nil)
	w = httptest.NewRecorder()
	ctx, err = api.jwtMiddleware(context.TODO(), w, req, config)
	require.NoError(t, err)
	ctx, err = api.authMaintainerMiddleware(ctx, w, req, config)
	assert.NoError(t, err, "no error should be returned because a jwt was given for an maintainer user")
	require.NotNil(t, getUserConsumer(ctx))
	assert.Equal(t, maintainer.ID, getUserConsumer(ctx).AuthConsumerUser.AuthentifiedUserID)

	req = assets.NewJWTAuthentifiedRequest(t, jwtAdmin, http.MethodGet, "", nil)
	w = httptest.NewRecorder()
	ctx, err = api.jwtMiddleware(context.TODO(), w, req, config)
	require.NoError(t, err)
	ctx, err = api.authMaintainerMiddleware(ctx, w, req, config)
	assert.NoError(t, err, "no error should be returned because a jwt was given for an admin user")
	require.NotNil(t, getUserConsumer(ctx))
	assert.Equal(t, admin.ID, getUserConsumer(ctx).AuthConsumerUser.AuthentifiedUserID)
}

func Test_authMiddleware_WithAuthConsumerScoped(t *testing.T) {
	api, db, _ := newTestAPI(t)

	g := assets.InsertGroup(t, db)
	u, _ := assets.InsertLambdaUser(t, db, g)
	localConsumer, err := authentication.LoadUserConsumerByTypeAndUserID(context.TODO(), db, sdk.ConsumerLocal, u.ID, authentication.LoadUserConsumerOptions.WithAuthentifiedUser)
	require.NoError(t, err)

	consumerOptions := builtin.NewConsumerOptions{
		Name:     "builtin",
		GroupIDs: []int64{g.ID},
		Scopes: []sdk.AuthConsumerScopeDetail{
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
		},
	}

	builtinConsumer, _, err := builtin.NewConsumer(context.TODO(), db, consumerOptions, localConsumer)
	require.NoError(t, err)
	builtinSession, err := authentication.NewSession(context.TODO(), db, &builtinConsumer.AuthConsumer, time.Second*5)
	require.NoError(t, err)
	jwt, err := authentication.NewSessionJWT(builtinSession, "")
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

func Test_authMiddlewareWithServiceOrWorker(t *testing.T) {
	api, db, router := newTestAPI(t)

	admin, jwtAdmin := assets.InsertAdminUser(t, db)

	config := &service.HandlerConfig{}

	// The token should be able to pass the authAdminMiddleware
	req := assets.NewJWTAuthentifiedRequest(t, jwtAdmin, http.MethodGet, "", nil)
	w := httptest.NewRecorder()
	ctx, err := api.jwtMiddleware(context.TODO(), w, req, config)
	require.NoError(t, err)
	_, err = api.authAdminMiddleware(ctx, w, req, config)
	require.NoError(t, err)
	_, err = api.authMaintainerMiddleware(ctx, w, req, config)
	require.NoError(t, err)

	// Admin create a consumer for a new service
	uri := api.Router.GetRoute(http.MethodPost, api.postConsumerByUserHandler, map[string]string{
		"permUsername": admin.Username,
	})
	require.NotEmpty(t, uri)
	req = assets.NewJWTAuthentifiedRequest(t, jwtAdmin, http.MethodPost, uri, sdk.AuthUserConsumer{
		AuthConsumer: sdk.AuthConsumer{
			Name:            sdk.RandomString(10),
			ValidityPeriods: sdk.NewAuthConsumerValidityPeriod(time.Now(), 0),
		},
		AuthConsumerUser: sdk.AuthUserConsumerData{
			ScopeDetails: sdk.NewAuthConsumerScopeDetails(sdk.AuthConsumerScopeService, sdk.AuthConsumerScopeAdmin),
		},
	})
	rec := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	require.Equal(t, 201, rec.Code)

	// Register a hatchery with the service consumer
	privateKey, err := jws.NewRandomRSAKey()
	require.NoError(t, err)
	publicKey, err := jws.ExportPublicKey(privateKey)
	require.NoError(t, err)
	hSrv := sdk.Service{
		CanonicalService: sdk.CanonicalService{
			Name:      sdk.RandomString(10),
			Type:      sdk.TypeHatchery,
			PublicKey: publicKey,
		},
	}

	var srvConsumer sdk.AuthConsumerCreateResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &srvConsumer))
	jwtHatchery := AuthentififyBuiltinConsumer(t, api, srvConsumer.Token, &hSrv)

	// The service token should not be able to pass the authAdminMiddleware or authMaintainerMiddleware anymore
	req = assets.NewJWTAuthentifiedRequest(t, jwtHatchery, http.MethodGet, "", nil)
	w = httptest.NewRecorder()
	ctx, err = api.jwtMiddleware(ctx, w, req, config)
	require.NoError(t, err)
	_, err = api.authAdminMiddleware(ctx, w, req, config)
	require.Error(t, err, "an error should be returned because the consumer is linked to a service")
	_, err = api.authMaintainerMiddleware(ctx, w, req, config)
	require.Error(t, err, "an error should be returned because the consumer is linked to a service")

	// Create a worker for the hatchery
	workflowTestContext := testRunWorkflow(t, api, router)
	require.NotNil(t, workflowTestContext.job)

	jwtWorkerSignin, err := hatchery.NewWorkerToken(hSrv.Name, privateKey, time.Now().Add(time.Hour), hatchery.SpawnArguments{
		HatcheryName: hSrv.Name,
		WorkerName:   hSrv.Name + "-worker",
		JobID:        fmt.Sprintf("%d", workflowTestContext.job.ID),
	})
	require.NoError(t, err)
	uri = api.Router.GetRoute("POST", api.postRegisterWorkerHandler, nil)
	require.NotEmpty(t, uri)
	req = assets.NewJWTAuthentifiedRequest(t, jwtWorkerSignin, "POST", uri, sdk.WorkerRegistrationForm{
		Arch:    runtime.GOARCH,
		OS:      runtime.GOOS,
		Version: sdk.VERSION,
	})
	rec = httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	require.Equal(t, 200, rec.Code)
	jwtWorker := rec.Header().Get("X-CDS-JWT")

	// The worker token should not be able to pass the authAdminMiddleware or authMaintainerMiddleware
	req = assets.NewJWTAuthentifiedRequest(t, jwtWorker, http.MethodGet, "", nil)
	w = httptest.NewRecorder()
	ctx, err = api.jwtMiddleware(context.TODO(), w, req, config)
	require.NoError(t, err)
	_, err = api.authAdminMiddleware(ctx, w, req, config)
	require.Error(t, err, "an error should be returned because the consumer is linked to a worker")
	_, err = api.authMaintainerMiddleware(ctx, w, req, config)
	require.Error(t, err, "an error should be returned because the consumer is linked to a worker")
}
