package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ovh/cds/engine/api/services"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/sdk"
)

func TestServicesHandlers(t *testing.T) {
	api, db, _ := newTestAPI(t)

	admin, jwtAdmin := assets.InsertAdminUser(t, db)
	lambda, jwtLambda := assets.InsertLambdaUser(t, db)

	s, _, jwtCDNService := assets.InitCDNService(t, db)
	t.Cleanup(func() { _ = services.Delete(db, s) })

	// Admin create a consumer for a new service
	uri := api.Router.GetRoute(http.MethodPost, api.postConsumerByUserHandler, map[string]string{
		"permUsername": admin.Username,
	})
	require.NotEmpty(t, uri)
	req := assets.NewJWTAuthentifiedRequest(t, jwtAdmin, http.MethodPost, uri, sdk.AuthUserConsumer{
		AuthConsumer: sdk.AuthConsumer{
			Name:            sdk.RandomString(10),
			ValidityPeriods: sdk.NewAuthConsumerValidityPeriod(time.Now(), 0),
		},
		AuthConsumerUser: sdk.AuthUserConsumerData{
			ScopeDetails: sdk.NewAuthConsumerScopeDetails(sdk.AuthConsumerScopeService),
		},
	})
	rec := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	require.Equal(t, 201, rec.Code)

	// Register a new service (srv1) for the admin's consumer
	var srv1 = sdk.Service{
		CanonicalService: sdk.CanonicalService{
			Name: sdk.RandomString(10),
			Type: sdk.TypeHatchery,
		},
	}

	// Signin with the new consumer
	var created sdk.AuthConsumerCreateResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &created))
	jwtSrv1 := AuthentififyBuiltinConsumer(t, api, created.Token, &srv1)

	// No CDN client should not be able to get the new service configuration
	uri = api.Router.GetRoute(http.MethodGet, api.getServiceHandler, map[string]string{
		"type": sdk.TypeHatchery,
	})
	require.NotEmpty(t, uri)
	req = assets.NewJWTAuthentifiedRequest(t, jwtAdmin, http.MethodGet, uri, nil)
	rec = httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	require.Equal(t, 403, rec.Code)

	// Service 1 should be able to post a heartbeat
	uri = api.Router.GetRoute(http.MethodPost, api.postServiceHearbeatHandler, nil)
	require.NotEmpty(t, uri)
	req = assets.NewJWTAuthentifiedRequest(t, jwtSrv1, http.MethodPost, uri, sdk.MonitoringStatus{})
	rec = httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	require.Equal(t, 204, rec.Code)

	// Lambda create a consumer for a new service
	uri = api.Router.GetRoute(http.MethodPost, api.postConsumerByUserHandler, map[string]string{
		"permUsername": lambda.Username,
	})
	require.NotEmpty(t, uri)
	req = assets.NewJWTAuthentifiedRequest(t, jwtLambda, http.MethodPost, uri, sdk.AuthUserConsumer{
		AuthConsumer: sdk.AuthConsumer{
			Name:            sdk.RandomString(10),
			ValidityPeriods: sdk.NewAuthConsumerValidityPeriod(time.Now(), 0),
		},
		AuthConsumerUser: sdk.AuthUserConsumerData{
			ScopeDetails: sdk.NewAuthConsumerScopeDetails(sdk.AuthConsumerScopeService),
		},
	})
	rec = httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	require.Equal(t, 201, rec.Code)

	// Register a new service (srv2) for the lambda's consumer
	var srv2 = sdk.Service{
		CanonicalService: sdk.CanonicalService{
			Name: sdk.RandomString(10),
			Type: sdk.TypeHatchery,
		},
	}

	// Signin with the new consumer
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &created))
	_ = AuthentififyBuiltinConsumer(t, api, created.Token, &srv2)

	// Lambda user should not be able to get service 2 config
	uri = api.Router.GetRoute(http.MethodGet, api.getServiceHandler, map[string]string{
		"type": sdk.TypeHatchery,
	})
	require.NotEmpty(t, uri)
	req = assets.NewJWTAuthentifiedRequest(t, jwtLambda, http.MethodGet, uri, nil)
	rec = httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	require.Equal(t, 403, rec.Code)

	// CDN should be able to get the both service 1 and 2 configuration
	uri = api.Router.GetRoute(http.MethodGet, api.getServiceHandler, map[string]string{
		"type": sdk.TypeHatchery,
	})
	require.NotEmpty(t, uri)
	req = assets.NewJWTAuthentifiedRequest(t, jwtCDNService, http.MethodGet, uri, nil)
	rec = httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	require.Equal(t, 200, rec.Code)
	var srvsConfig []sdk.ServiceConfiguration
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &srvsConfig))
	require.True(t, len(srvsConfig) >= 1)
	var srv1Found, srv2Found bool
	for _, s := range srvsConfig {
		if s.Name == srv1.Name {
			srv1Found = true
		}
		if s.Name == srv2.Name {
			srv2Found = true
		}
		if srv1Found && srv2Found {
			break
		}
	}
	require.True(t, srv1Found && srv2Found, "service srv1 and srv2 configs should be returned to admin user")
}
