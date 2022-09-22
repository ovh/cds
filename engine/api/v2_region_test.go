package api

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/ovh/cds/engine/api/rbac"
	"github.com/rockbears/yaml"
	"net/http/httptest"
	"testing"

	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/sdk"
	"github.com/stretchr/testify/require"
)

func Test_crudRegion(t *testing.T) {
	api, db, _ := newTestAPI(t)

	u, pass := assets.InsertLambdaUser(t, db)

	// Insert rbac
	perm := fmt.Sprintf(`name: perm-region-%s
globals:
  - role: %s
    users: [%s]
`, sdk.RandomString(10), sdk.GlobalRoleManageRegion, u.Username)

	var rb sdk.RBAC
	require.NoError(t, yaml.Unmarshal([]byte(perm), &rb))
	rb.Globals[0].RBACUsersIDs = []string{u.ID}
	require.NoError(t, rbac.Insert(context.TODO(), db, &rb))

	reg := sdk.Region{Name: sdk.RandomString(10)}

	uri := api.Router.GetRouteV2("POST", api.postRegionHandler, nil)
	test.NotEmpty(t, uri)
	req := assets.NewAuthentifiedRequest(t, u, pass, "POST", uri, &reg)
	w := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w, req)
	require.Equal(t, 201, w.Code)

	// Then Get the region
	uriGet := api.Router.GetRouteV2("GET", api.getRegionHandler, map[string]string{"regionIdentifier": reg.Name})
	test.NotEmpty(t, uriGet)
	reqGet := assets.NewAuthentifiedRequest(t, u, pass, "GET", uriGet, nil)
	wGet := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(wGet, reqGet)
	require.Equal(t, 200, wGet.Code)

	var regionGet sdk.Organization
	require.NoError(t, json.Unmarshal(wGet.Body.Bytes(), &regionGet))
	require.Equal(t, regionGet.Name, regionGet.Name)

	// Then Delete Region
	uriDelete := api.Router.GetRouteV2("DELETE", api.deleteRegionHandler, map[string]string{"regionIdentifier": reg.Name})
	test.NotEmpty(t, uriDelete)
	reqDelete := assets.NewAuthentifiedRequest(t, u, pass, "DELETE", uriDelete, nil)
	wDelete := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(wDelete, reqDelete)
	require.Equal(t, 204, wDelete.Code)

	// Then check if region has been deleted
	uriList := api.Router.GetRouteV2("GET", api.getRegionsHandler, nil)
	test.NotEmpty(t, uriList)
	wList := httptest.NewRecorder()
	reqList := assets.NewAuthentifiedRequest(t, u, pass, "GET", uriList, nil)
	api.Router.Mux.ServeHTTP(wList, reqList)
	require.Equal(t, 200, wList.Code)

	var regions []sdk.Region
	require.NoError(t, json.Unmarshal(wList.Body.Bytes(), &regions))
	require.Len(t, regions, 0)
}
