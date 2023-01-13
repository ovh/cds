package api

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/ovh/cds/engine/api/rbac"
	"io"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/sdk"
	"github.com/stretchr/testify/require"
)

func Test_crudRbacHandler(t *testing.T) {
	api, db, _ := newTestAPI(t)

	_, err := db.Exec("DELETE FROM rbac")
	require.NoError(t, err)

	p := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))
	u, pass := assets.InsertAdminUser(t, db)
	g := assets.InsertTestGroup(t, db, sdk.RandomString(10))

	vars := map[string]string{}
	uri := api.Router.GetRouteV2("POST", api.postImportRBACHandler, vars)
	test.NotEmpty(t, uri)
	req := assets.NewAuthentifiedRequest(t, u, pass, "POST", uri, nil)

	body := fmt.Sprintf(`name: perm-test
projects:
  - role: read
    projects: [%s]
    users: [%s]
    groups: [%s]
global:
 - role: manage-permission
   users: [%s]
   groups: [%s]
`, p.Key, u.Username, g.Name, u.Username, g.Name)

	// Here, we insert the vcs server as a CDS administrator
	req.Body = io.NopCloser(strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-yaml")

	w := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w, req)
	require.Equal(t, 201, w.Code)

	// GET RBAC
	varsGET := map[string]string{"rbacIdentifier": "perm-test"}
	uriGET := api.Router.GetRouteV2("GET", api.getRBACHandler, varsGET)
	test.NotEmpty(t, uriGET)
	reqGET := assets.NewAuthentifiedRequest(t, u, pass, "GET", uriGET, nil)

	wGET := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(wGET, reqGET)
	require.Equal(t, 200, wGET.Code)
	var rbacGET sdk.RBAC
	require.NoError(t, json.Unmarshal(wGET.Body.Bytes(), &rbacGET))

	require.Equal(t, 1, len(rbacGET.Global))
	require.Equal(t, "manage-permission", rbacGET.Global[0].Role)
	require.Equal(t, 1, len(rbacGET.Global[0].RBACUsersName))
	require.Equal(t, 1, len(rbacGET.Global[0].RBACGroupsName))
	require.Equal(t, u.Username, rbacGET.Global[0].RBACUsersName[0])
	require.Equal(t, g.Name, rbacGET.Global[0].RBACGroupsName[0])

	require.Equal(t, 1, len(rbacGET.Projects))
	require.Equal(t, "read", rbacGET.Projects[0].Role)
	require.Equal(t, 1, len(rbacGET.Projects[0].RBACUsersName))
	require.Equal(t, 1, len(rbacGET.Projects[0].RBACGroupsName))
	require.Equal(t, u.Username, rbacGET.Projects[0].RBACUsersName[0])
	require.Equal(t, g.Name, rbacGET.Projects[0].RBACGroupsName[0])

	// Delete
	varsDelete := map[string]string{"rbacIdentifier": rbacGET.ID}
	uriDelete := api.Router.GetRouteV2("DELETE", api.deleteRBACHandler, varsDelete)
	test.NotEmpty(t, uriDelete)
	reqDelete := assets.NewAuthentifiedRequest(t, u, pass, "DELETE", uriDelete, nil)

	wDelete := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(wDelete, reqDelete)
	require.Equal(t, 200, wDelete.Code)

	_, err = rbac.LoadRBACByID(context.TODO(), db, rbacGET.ID)
	require.True(t, sdk.ErrorIs(err, sdk.ErrNotFound))
}
