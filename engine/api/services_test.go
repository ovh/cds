package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ovh/cds/engine/api/bootstrap"
	"github.com/ovh/cds/engine/api/services"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/sdk"
)

func TestServicesHandlers(t *testing.T) {
	api, _, _, end := newTestAPI(t, bootstrap.InitiliazeDB)
	defer end()

	admin, jwtRaw := assets.InsertAdminUser(api.mustDB())

	data := sdk.AuthConsumer{
		Name:   sdk.RandomString(10),
		Scopes: []sdk.AuthConsumerScope{sdk.AuthConsumerScopeService},
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
			Type: services.TypeHatchery,
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

	require.NoError(t, services.Delete(api.mustDB(), &srv))

}
