package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/rockbears/yaml"
	"github.com/stretchr/testify/require"

	"github.com/ovh/cds/engine/api/authentication"
	"github.com/ovh/cds/engine/api/authentication/hatchery"
	hatch "github.com/ovh/cds/engine/api/hatchery"
	"github.com/ovh/cds/engine/api/rbac"
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
	consumer, err := authentication.LoadHatcheryConsumerByHatcheryID(context.TODO(), db, hatcheryCreated.ID)
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

}

func Test_crudHatchery(t *testing.T) {
	api, db, _ := newTestAPI(t)

	db.Exec("DELETE FROM hatchery")

	u, pass := assets.InsertLambdaUser(t, db)

	// Insert rbac
	perm := fmt.Sprintf(`name: perm-hatchery-%s
globals:
  - role: %s
    users: [%s]
`, sdk.RandomString(10), sdk.GlobalRoleManageHatchery, u.Username)

	var rb sdk.RBAC
	require.NoError(t, yaml.Unmarshal([]byte(perm), &rb))
	rb.Globals[0].RBACUsersIDs = []string{u.ID}
	require.NoError(t, rbac.Insert(context.TODO(), db, &rb))

	h := sdk.Hatchery{Name: sdk.RandomString(10)}

	uri := api.Router.GetRouteV2("POST", api.postHatcheryHandler, nil)
	test.NotEmpty(t, uri)
	req := assets.NewAuthentifiedRequest(t, u, pass, "POST", uri, &h)
	w := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w, req)
	require.Equal(t, 201, w.Code)
	var hatcheryCreated sdk.Hatchery
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &hatcheryCreated))

	// Then Get the hatchery
	uriGet := api.Router.GetRouteV2("GET", api.getHatcheryHandler, map[string]string{"hatcheryIdentifier": h.Name})
	test.NotEmpty(t, uriGet)
	reqGet := assets.NewAuthentifiedRequest(t, u, pass, "GET", uriGet, nil)
	wGet := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(wGet, reqGet)
	require.Equal(t, 200, wGet.Code)

	var hatcheryGet sdk.Hatchery
	require.NoError(t, json.Unmarshal(wGet.Body.Bytes(), &hatcheryGet))
	require.Equal(t, hatcheryGet.Name, hatcheryGet.Name)

	// Login with hatchery
	uriLogin := api.Router.GetRouteV2("POST", api.postAuthHatcherySigninHandler, nil)
	test.NotEmpty(t, uriLogin)

	signinRequest := sdk.AuthConsumerHatcherySigninRequest{
		Token:   hatcheryCreated.Token,
		Name:    hatcheryCreated.Name,
		HTTPURL: "local.host",
	}
	reqSignin := assets.NewRequest(t, "POST", uriLogin, &signinRequest)
	wSignin := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(wSignin, reqSignin)
	require.Equal(t, 200, wSignin.Code)

	var authSigninResponse sdk.AuthConsumerHatcherySigninResponse
	require.NoError(t, json.Unmarshal(wSignin.Body.Bytes(), &authSigninResponse))
	require.NotEmpty(t, authSigninResponse.Token)

	// Then Delete hatchery
	uriDelete := api.Router.GetRouteV2("DELETE", api.deleteHatcheryHandler, map[string]string{"hatcheryIdentifier": h.Name})
	test.NotEmpty(t, uriDelete)
	reqDelete := assets.NewAuthentifiedRequest(t, u, pass, "DELETE", uriDelete, nil)
	wDelete := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(wDelete, reqDelete)
	require.Equal(t, 204, wDelete.Code)

	// Then check if hatchery has been deleted
	uriList := api.Router.GetRouteV2("GET", api.getHatcheriesHandler, nil)
	test.NotEmpty(t, uriList)
	wList := httptest.NewRecorder()
	reqList := assets.NewAuthentifiedRequest(t, u, pass, "GET", uriList, nil)
	api.Router.Mux.ServeHTTP(wList, reqList)
	require.Equal(t, 200, wList.Code)

	var hs []sdk.Hatchery
	require.NoError(t, json.Unmarshal(wList.Body.Bytes(), &hs))
	require.Len(t, hs, 0)
}
