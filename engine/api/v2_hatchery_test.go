package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ovh/cds/engine/api/authentication"
	"github.com/ovh/cds/engine/api/authentication/hatchery"
	hatch "github.com/ovh/cds/engine/api/hatchery"
	"github.com/ovh/cds/engine/api/rbac"
	"github.com/ovh/cds/engine/api/region"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/sdk"
)

func Test_hatcheryHeartbeat(t *testing.T) {
	api, db, _ := newTestAPI(t)

	db.Exec("DELETE FROM hatchery")

	u, pass := assets.InsertAdminUser(t, db)

	// CREATE HATCHERY
	h := sdk.Hatchery{Name: sdk.RandomString(10)}
	uri := api.Router.GetRouteV2("POST", api.postHatcheryHandler, nil)
	test.NotEmpty(t, uri)
	req := assets.NewAuthentifiedRequest(t, u, pass, "POST", uri, &h)
	w := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w, req)
	require.Equal(t, 201, w.Code)
	var hatcheryCreated sdk.Hatchery
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &hatcheryCreated))

	// GET CONSUMER AND CREATE SESSION
	consumer, err := authentication.LoadHatcheryConsumerByName(context.TODO(), db, hatcheryCreated.Name)
	require.NoError(t, err)
	session, err := authentication.NewSession(context.TODO(), db, &consumer.AuthConsumer, hatchery.SessionDuration)
	require.NoError(t, err)
	jwt, err := authentication.NewSessionJWT(session, "")
	require.NoError(t, err)

	// Post heartbeat
	now := time.Now()
	uriHeartbeat := api.Router.GetRouteV2("POST", api.postHatcheryHeartbeatHandler, nil)
	heartBeatRequest := sdk.MonitoringStatus{
		Now: now,
	}
	reqHeartbeat := assets.NewJWTAuthentifiedRequest(t, jwt, "POST", uriHeartbeat, &heartBeatRequest)
	wHeartbeat := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(wHeartbeat, reqHeartbeat)
	require.Equal(t, 204, wHeartbeat.Code)

	hatcheryStatus, err := hatch.LoadHatcheryStatusByHatcheryID(context.TODO(), db, hatcheryCreated.ID)
	require.NoError(t, err)
	require.True(t, now.Equal(hatcheryStatus.Status.Now))

	// Regen token with hatchery auth
	uriRegen := api.Router.GetRouteV2("POST", api.postHatcheryRegenTokenHandler, map[string]string{"hatcheryIdentifier": h.Name})
	reqRegen := assets.NewJWTAuthentifiedRequest(t, jwt, "POST", uriRegen, nil)
	wRegen := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(wRegen, reqRegen)
	require.Equal(t, 200, wRegen.Code)

	var hatchResp sdk.HatcheryGetResponse
	require.NoError(t, json.Unmarshal(wRegen.Body.Bytes(), &hatchResp))

	require.NotEmpty(t, hatchResp.Token)
}

func Test_crudHatchery(t *testing.T) {
	api, db, _ := newTestAPI(t)

	u, jwt := assets.InsertLambdaUser(t, db)

	// Insert rbac for user to manage hatchery
	require.NoError(t, rbac.Insert(context.TODO(), db, &sdk.RBAC{
		Name: "perm-global-" + sdk.RandomString(10),
		Global: []sdk.RBACGlobal{
			{
				Role:          sdk.GlobalRoleManageHatchery,
				RBACUsersIDs:  []string{u.ID},
				RBACUsersName: []string{u.Username},
			},
		},
	}))

	// Create Hatchery
	uri := api.Router.GetRouteV2(http.MethodPost, api.postHatcheryHandler, nil)
	require.NotEmpty(t, uri)
	req := assets.NewJWTAuthentifiedRequest(t, jwt, http.MethodPost, uri, &sdk.Hatchery{
		Name: sdk.RandomString(10),
	})
	rec := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusCreated, rec.Code)
	var hatcheryCreated sdk.HatcheryGetResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &hatcheryCreated))

	// Create RBAC for hatchery
	reg := sdk.Region{Name: sdk.RandomString(10)}
	require.NoError(t, region.Insert(context.TODO(), db, &reg))
	require.NoError(t, rbac.Insert(context.TODO(), db, &sdk.RBAC{
		Name: "perm-hatchery-" + hatcheryCreated.Name,
		Hatcheries: []sdk.RBACHatchery{
			{
				Role:         sdk.HatcheryRoleSpawn,
				HatcheryName: hatcheryCreated.Name,
				HatcheryID:   hatcheryCreated.ID,
				RegionName:   reg.Name,
				RegionID:     reg.ID,
			},
		},
	}))

	// Then Get the hatchery
	uri = api.Router.GetRouteV2(http.MethodGet, api.getHatcheryHandler, map[string]string{
		"hatcheryIdentifier": hatcheryCreated.Name,
	})
	require.NotEmpty(t, uri)
	req = assets.NewJWTAuthentifiedRequest(t, jwt, http.MethodGet, uri, nil)
	rec = httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
	var getResponse sdk.Hatchery
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &getResponse))
	require.Equal(t, hatcheryCreated.Name, getResponse.Name)

	// Regen hatchery token
	uriRegen := api.Router.GetRouteV2(http.MethodPost, api.postHatcheryRegenTokenHandler, map[string]string{
		"hatcheryIdentifier": hatcheryCreated.Name,
	})
	req = assets.NewJWTAuthentifiedRequest(t, jwt, http.MethodPost, uriRegen, nil)
	rec = httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	require.Equal(t, 200, rec.Code)
	var regenResponse sdk.HatcheryGetResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &regenResponse))
	require.NotEmpty(t, regenResponse.Token)

	// Then Delete hatchery
	uri = api.Router.GetRouteV2(http.MethodDelete, api.deleteHatcheryHandler, map[string]string{
		"hatcheryIdentifier": hatcheryCreated.Name,
	})
	require.NotEmpty(t, uri)
	req = assets.NewJWTAuthentifiedRequest(t, jwt, http.MethodDelete, uri, nil)
	rec = httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	require.Equal(t, 204, rec.Code)

	// Then check if hatchery has been deleted
	uri = api.Router.GetRouteV2(http.MethodGet, api.getHatcheriesHandler, nil)
	require.NotEmpty(t, uri)
	rec = httptest.NewRecorder()
	req = assets.NewJWTAuthentifiedRequest(t, jwt, http.MethodGet, uri, nil)
	api.Router.Mux.ServeHTTP(rec, req)
	require.Equal(t, 200, rec.Code)
	var listResponse []sdk.Hatchery
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &listResponse))
	require.NotContainsf(t, listResponse, func(h sdk.Hatchery) bool { return h.Name == hatcheryCreated.Name }, "Hatchery %s should have been deleted", hatcheryCreated.Name)

	// Check rbac was deleted
	_, err := rbac.LoadRBACByName(context.TODO(), db, "perm-hatchery-"+hatcheryCreated.Name, rbac.LoadOptions.All)
	require.Error(t, err)
	require.True(t, sdk.ErrorIs(err, sdk.ErrNotFound))
}
