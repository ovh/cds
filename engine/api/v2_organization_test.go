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

func Test_crudOrganization(t *testing.T) {
	api, db, _ := newTestAPI(t)

	u, pass := assets.InsertLambdaUser(t, db)

	// Insert rbac
	perm := fmt.Sprintf(`name: perm-orga-%s
global:
  - role: %s
    users: [%s]
`, sdk.RandomString(10), sdk.GlobalRoleManageOrganization, u.Username)

	var rb sdk.RBAC
	require.NoError(t, yaml.Unmarshal([]byte(perm), &rb))
	rb.Global[0].RBACUsersIDs = []string{u.ID}
	require.NoError(t, rbac.Insert(context.TODO(), db, &rb))

	orga := sdk.Organization{Name: sdk.RandomString(10)}

	uri := api.Router.GetRouteV2("POST", api.postOrganizationHandler, nil)
	test.NotEmpty(t, uri)
	req := assets.NewAuthentifiedRequest(t, u, pass, "POST", uri, &orga)
	w := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w, req)
	require.Equal(t, 201, w.Code)

	// Then Get the organization
	uriGet := api.Router.GetRouteV2("GET", api.getOrganizationHandler, map[string]string{"organizationIdentifier": orga.Name})
	test.NotEmpty(t, uriGet)
	reqGet := assets.NewAuthentifiedRequest(t, u, pass, "GET", uriGet, nil)
	wGet := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(wGet, reqGet)
	require.Equal(t, 200, wGet.Code)

	var orgaGet sdk.Organization
	require.NoError(t, json.Unmarshal(wGet.Body.Bytes(), &orgaGet))
	require.Equal(t, orga.Name, orgaGet.Name)

	// Then Delete Organization
	uriDelete := api.Router.GetRouteV2("DELETE", api.deleteOrganizationHandler, map[string]string{"organizationIdentifier": orga.Name})
	test.NotEmpty(t, uriDelete)
	reqDelete := assets.NewAuthentifiedRequest(t, u, pass, "DELETE", uriDelete, nil)
	wDelete := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(wDelete, reqDelete)
	require.Equal(t, 204, wDelete.Code)

	// Then check if organization has been deleted
	uriList := api.Router.GetRouteV2("GET", api.getOrganizationsHandler, nil)
	test.NotEmpty(t, uriList)
	wList := httptest.NewRecorder()
	reqList := assets.NewAuthentifiedRequest(t, u, pass, "GET", uriList, nil)
	api.Router.Mux.ServeHTTP(wList, reqList)
	require.Equal(t, 200, wList.Code)

	var orgs []sdk.Organization
	require.NoError(t, json.Unmarshal(wList.Body.Bytes(), &orgs))
	require.Len(t, orgs, 1)
	require.Equal(t, "default", orgs[0].Name)
}
