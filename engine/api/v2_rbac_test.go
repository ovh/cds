package api

import (
	"context"
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

func Test_postImportAndDeleteRbacHandler(t *testing.T) {
	api, db, _ := newTestAPI(t)

	_, err := db.Exec("DELETE FROM rbac")
	require.NoError(t, err)

	p := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))
	u, pass := assets.InsertAdminUser(t, db)
	g := assets.InsertTestGroup(t, db, sdk.RandomString(10))

	vars := map[string]string{}
	uri := api.Router.GetRouteV2("POST", api.postImportRbacHandler, vars)
	test.NotEmpty(t, uri)
	req := assets.NewAuthentifiedRequest(t, u, pass, "POST", uri, nil)

	body := fmt.Sprintf(`name: perm-test
projects:
  - role: read
    projects: [%s]
    users: [%s]
    groups: [%s]
globals:
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

	rbacDB, err := rbac.LoadRBACByName(context.TODO(), db, "perm-test", rbac.LoadOptions.LoadRBACProject, rbac.LoadOptions.LoadRBACGlobal)
	require.NoError(t, err)

	require.Equal(t, 1, len(rbacDB.Globals))
	require.Equal(t, "manage-permission", rbacDB.Globals[0].Role)
	require.Equal(t, 1, len(rbacDB.Globals[0].RBACUsersIDs))
	require.Equal(t, 1, len(rbacDB.Globals[0].RBACGroupsIDs))
	require.Equal(t, u.ID, rbacDB.Globals[0].RBACUsersIDs[0])
	require.Equal(t, g.ID, rbacDB.Globals[0].RBACGroupsIDs[0])

	require.Equal(t, 1, len(rbacDB.Projects))
	require.Equal(t, "read", rbacDB.Projects[0].Role)
	require.Equal(t, 1, len(rbacDB.Projects[0].RBACUsersIDs))
	require.Equal(t, 1, len(rbacDB.Projects[0].RBACGroupsIDs))
	require.Equal(t, u.ID, rbacDB.Projects[0].RBACUsersIDs[0])
	require.Equal(t, g.ID, rbacDB.Projects[0].RBACGroupsIDs[0])

	// Delete
	varsDelete := map[string]string{"rbacIdentifier": rbacDB.ID}
	uriDelete := api.Router.GetRouteV2("DELETE", api.deleteRbacHandler, varsDelete)
	test.NotEmpty(t, uriDelete)
	reqDelete := assets.NewAuthentifiedRequest(t, u, pass, "DELETE", uriDelete, nil)

	wDelete := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(wDelete, reqDelete)
	require.Equal(t, 200, wDelete.Code)

	_, err = rbac.LoadRBACByID(context.TODO(), db, rbacDB.ID)
	require.True(t, sdk.ErrorIs(err, sdk.ErrNotFound))
}
