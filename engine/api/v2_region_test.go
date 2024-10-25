package api

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/ovh/cds/engine/api/hatchery"
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

	db.Exec("DELETE FROM region")

	u, pass := assets.InsertLambdaUser(t, db)

	hatch := sdk.Hatchery{Name: sdk.RandomString(10)}
	require.NoError(t, hatchery.Insert(context.TODO(), db, &hatch))

	// Insert rbac
	perm := fmt.Sprintf(`name: perm-region-%s
global:
  - role: %s
    users: [%s]
`, sdk.RandomString(10), sdk.GlobalRoleManageRegion, u.Username)

	var rb sdk.RBAC
	require.NoError(t, yaml.Unmarshal([]byte(perm), &rb))
	rb.Global[0].RBACUsersIDs = []string{u.ID}
	require.NoError(t, rbac.Insert(context.TODO(), db, &rb))

	reg := sdk.Region{Name: sdk.RandomString(10)}

	uri := api.Router.GetRouteV2("POST", api.postRegionHandler, nil)
	test.NotEmpty(t, uri)
	req := assets.NewAuthentifiedRequest(t, u, pass, "POST", uri, &reg)
	w := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w, req)
	require.Equal(t, 201, w.Code)

	//regDB, err := region.LoadRegionByName(context.TODO(), api.mustDB(), reg.Name)
	//require.NoError(t, err)

	rbacReadRegion := `name: perm-%s
regions:
- role: %s
  region: %s
  users: [%s]
  organizations: [default]
hatcheries:
- role: %s
  region: %s
  hatchery: %s`

	rbacReadRegion = fmt.Sprintf(rbacReadRegion, sdk.RandomString(10), sdk.RegionRoleList, reg.Name, u.Username,
		sdk.HatcheryRoleSpawn, reg.Name, hatch.Name)
	var rbRead sdk.RBAC
	require.NoError(t, yaml.Unmarshal([]byte(rbacReadRegion), &rbRead))

	rLoader := NewRBACLoader(db)
	rLoader.FillRBACWithIDs(context.TODO(), &rbRead)

	require.NoError(t, rbac.Insert(context.TODO(), db, &rbRead))

	// Then Get the region
	uriGet := api.Router.GetRouteV2("GET", api.getRegionHandler, map[string]string{"regionIdentifier": reg.Name})
	test.NotEmpty(t, uriGet)
	reqGet := assets.NewAuthentifiedRequest(t, u, pass, "GET", uriGet, nil)
	wGet := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(wGet, reqGet)
	require.Equal(t, 200, wGet.Code)

	var regionGet sdk.Region
	require.NoError(t, json.Unmarshal(wGet.Body.Bytes(), &regionGet))
	require.Equal(t, regionGet.Name, regionGet.Name)

	// remove rbac before deleting region
	require.NoError(t, rbac.Delete(context.TODO(), db, rbRead))

	// Then Delete Region
	uriDelete := api.Router.GetRouteV2("DELETE", api.deleteRegionHandler, map[string]string{"regionIdentifier": reg.Name})
	test.NotEmpty(t, uriDelete)
	reqDelete := assets.NewAuthentifiedRequest(t, u, pass, "DELETE", uriDelete, nil)
	wDelete := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(wDelete, reqDelete)
	require.Equal(t, 204, wDelete.Code)

	// check rbac deletion
	_, err := rbac.LoadRBACByName(context.TODO(), db, rbRead.Name, rbac.LoadOptions.All)
	require.Error(t, err)
	require.True(t, sdk.ErrorIs(err, sdk.ErrNotFound))

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
