package api

import (
	"context"
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
	api, _, _ := newTestAPI(t)

	admin, jwtRaw := assets.InsertAdminUser(t, api.mustDB())
	_, jwtLambda := assets.InsertLambdaUser(t, api.mustDB())

	data := sdk.AuthConsumer{
		Name:         sdk.RandomString(10),
		ScopeDetails: sdk.NewAuthConsumerScopeDetails(sdk.AuthConsumerScopeService),
		IssuedAt:     time.Now(),
	}

	uri := api.Router.GetRoute(http.MethodPost, api.postConsumerByUserHandler, map[string]string{
		"permUsername": admin.Username,
	})
	require.NotEmpty(t, uri)
	req := assets.NewJWTAuthentifiedRequest(t, jwtRaw, http.MethodPost, uri, data)
	rec := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	require.Equal(t, 201, rec.Code)

	var created sdk.AuthConsumerCreateResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &created))

	signinToken := created.Token

	jwtSrv := AuthentififyBuiltinConsumer(t, api, signinToken)

	var srv = sdk.Service{
		CanonicalService: sdk.CanonicalService{
			Name: sdk.RandomString(10),
			Type: sdk.TypeHatchery,
		},
	}

	uri = api.Router.GetRoute(http.MethodPost, api.postServiceRegisterHandler, nil)
	require.NotEmpty(t, uri)
	req = assets.NewJWTAuthentifiedRequest(t, jwtSrv, http.MethodPost, uri, srv)
	rec = httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	require.Equal(t, 200, rec.Code)

	allSrv, err := services.LoadAll(context.Background(), api.mustDB())
	require.NoError(t, err)
	require.True(t, len(allSrv) >= 1)

	var srvFound bool
	for _, s := range allSrv {
		if s.Name == srv.Name {
			srvFound = true
			t.Logf("service: %+v", s)
			srv = s
			break
		}
	}
	require.True(t, srvFound, "service not found")

	// Post a heartbeat

	var mon sdk.MonitoringStatus
	uri = api.Router.GetRoute(http.MethodPost, api.postServiceHearbeatHandler, nil)
	require.NotEmpty(t, uri)
	req = assets.NewJWTAuthentifiedRequest(t, jwtSrv, http.MethodPost, uri, mon)
	rec = httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	require.Equal(t, 204, rec.Code)

	// Get service with lambda user => 404
	uri = api.Router.GetRoute(http.MethodGet, api.getServiceHandler, map[string]string{
		"type": sdk.TypeHatchery,
	})
	require.NotEmpty(t, uri)
	req = assets.NewJWTAuthentifiedRequest(t, jwtLambda, http.MethodGet, uri, data)
	rec = httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	require.Equal(t, 404, rec.Code)

	// lambda user Insert a service
	uri = api.Router.GetRoute(http.MethodPost, api.postServiceRegisterHandler, nil)
	require.NotEmpty(t, uri)
	req = assets.NewJWTAuthentifiedRequest(t, jwtLambda, http.MethodPost, uri, srv)
	rec = httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	require.Equal(t, 200, rec.Code)

	// Get service with lambda user => 404
	uri = api.Router.GetRoute(http.MethodGet, api.getServiceHandler, map[string]string{
		"type": sdk.TypeHatchery,
	})
	require.NotEmpty(t, uri)
	req = assets.NewJWTAuthentifiedRequest(t, jwtLambda, http.MethodGet, uri, data)
	rec = httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	require.Equal(t, 200, rec.Code)

	var servs []sdk.ServiceConfiguration
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &servs))
	require.Equal(t, 1, len(servs))

	require.NoError(t, services.Delete(api.mustDB(), &srv))
}
