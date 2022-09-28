package api

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/ovh/cds/engine/api/organization"
	"github.com/ovh/cds/engine/api/rbac"
	"github.com/ovh/cds/engine/api/region"
	"github.com/rockbears/yaml"
	"net/http/httptest"
	"testing"

	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/sdk"
	"github.com/stretchr/testify/require"
)

func Test_crudOrganizationRegion(t *testing.T) {
	api, db, _ := newTestAPI(t)

	u, pass := assets.InsertLambdaUser(t, db)

	// Insert rbac
	perm := fmt.Sprintf(`name: perm-orga-%s
globals:
  - role: %s
    users: [%s]
`, sdk.RandomString(10), sdk.GlobalRoleManageOrganization, u.Username)

	var rb sdk.RBAC
	require.NoError(t, yaml.Unmarshal([]byte(perm), &rb))
	rb.Globals[0].RBACUsersIDs = []string{u.ID}
	require.NoError(t, rbac.Insert(context.TODO(), db, &rb))

	orga := sdk.Organization{Name: sdk.RandomString(10)}
	require.NoError(t, organization.Insert(context.TODO(), db, &orga))

	reg := sdk.Region{Name: sdk.RandomString(10)}
	require.NoError(t, region.Insert(context.TODO(), db, &reg))

	uri := api.Router.GetRouteV2("POST", api.postAllowRegionOnOrganizationHandler, map[string]string{"organizationIdentifier": orga.Name})
	test.NotEmpty(t, uri)
	req := assets.NewAuthentifiedRequest(t, u, pass, "POST", uri, &sdk.Region{Name: reg.Name})
	w := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w, req)
	require.Equal(t, 201, w.Code)

	// Then list region allowed in the organization
	uriGet := api.Router.GetRouteV2("GET", api.getListRegionAllowedOnIrganizationHandler, map[string]string{"organizationIdentifier": orga.Name})
	test.NotEmpty(t, uriGet)
	reqGet := assets.NewAuthentifiedRequest(t, u, pass, "GET", uriGet, nil)
	wGet := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(wGet, reqGet)
	require.Equal(t, 200, wGet.Code)

	var regs []sdk.Region
	require.NoError(t, json.Unmarshal(wGet.Body.Bytes(), &regs))
	require.Len(t, regs, 1)
	require.Equal(t, reg.Name, regs[0].Name)

	// Then Delete Organization
	uriDelete := api.Router.GetRouteV2("DELETE", api.deleteRegionFromOrganizationHandler, map[string]string{"organizationIdentifier": orga.Name, "regionIdentifier": reg.Name})
	test.NotEmpty(t, uriDelete)
	reqDelete := assets.NewAuthentifiedRequest(t, u, pass, "DELETE", uriDelete, nil)
	wDelete := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(wDelete, reqDelete)
	require.Equal(t, 204, wDelete.Code)

	// Then check if organization has been deleted
	uriList := api.Router.GetRouteV2("GET", api.getListRegionAllowedOnIrganizationHandler, map[string]string{"organizationIdentifier": orga.Name})
	test.NotEmpty(t, uriList)
	wList := httptest.NewRecorder()
	reqList := assets.NewAuthentifiedRequest(t, u, pass, "GET", uriList, nil)
	api.Router.Mux.ServeHTTP(wList, reqList)
	require.Equal(t, 200, wList.Code)

	var regs2 []sdk.Region
	require.NoError(t, json.Unmarshal(wList.Body.Bytes(), &regs2))
	require.Len(t, regs2, 0)
}
