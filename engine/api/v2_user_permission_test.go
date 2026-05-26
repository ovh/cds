package api

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/ovh/cds/engine/api/rbac"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/sdk"
	"github.com/stretchr/testify/require"
)

func Test_getUserPermissionHandler_noPermissions(t *testing.T) {
	api, db, _ := newTestAPI(t)

	_, err := db.Exec("DELETE FROM rbac")
	require.NoError(t, err)

	u, pass := assets.InsertAdminUser(t, db)

	vars := map[string]string{"user": u.Username}
	uri := api.Router.GetRouteV2("GET", api.getUserPermissionHandler, vars)
	test.NotEmpty(t, uri)

	req := assets.NewAuthentifiedRequest(t, u, pass, "GET", uri, nil)
	w := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w, req)
	require.Equal(t, 200, w.Code)

	var summary sdk.PermissionSummary
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &summary))

	require.Empty(t, summary.Global)
	require.Empty(t, summary.Projects)
	require.Empty(t, summary.Regions)
}

func Test_getUserPermissionHandler_withGlobalRole(t *testing.T) {
	api, db, _ := newTestAPI(t)

	_, err := db.Exec("DELETE FROM rbac")
	require.NoError(t, err)

	u, pass := assets.InsertAdminUser(t, db)

	rb := sdk.RBAC{
		Name: sdk.RandomString(10),
		Global: []sdk.RBACGlobal{
			{
				Role:         sdk.GlobalRoleManagePermission,
				RBACUsersIDs: []string{u.ID},
			},
		},
	}
	require.NoError(t, rbac.Insert(context.TODO(), db, &rb))
	t.Cleanup(func() {
		db.Exec("DELETE FROM rbac WHERE id = $1", rb.ID)
	})

	vars := map[string]string{"user": u.Username}
	uri := api.Router.GetRouteV2("GET", api.getUserPermissionHandler, vars)
	test.NotEmpty(t, uri)

	req := assets.NewAuthentifiedRequest(t, u, pass, "GET", uri, nil)
	w := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w, req)
	require.Equal(t, 200, w.Code)

	var summary sdk.PermissionSummary
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &summary))

	require.Len(t, summary.Global, 1)
	require.Contains(t, summary.Global, sdk.GlobalRoleManagePermission)
	require.Empty(t, summary.Projects)
	require.Empty(t, summary.Regions)
}

func Test_getUserPermissionHandler_withProjectRole(t *testing.T) {
	api, db, _ := newTestAPI(t)

	_, err := db.Exec("DELETE FROM rbac")
	require.NoError(t, err)

	p := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))
	u, pass := assets.InsertAdminUser(t, db)

	rb := sdk.RBAC{
		Name: sdk.RandomString(10),
		Projects: []sdk.RBACProject{
			{
				Role:            sdk.ProjectRoleRead,
				RBACProjectKeys: []string{p.Key},
				RBACUsersIDs:    []string{u.ID},
			},
		},
	}
	require.NoError(t, rbac.Insert(context.TODO(), db, &rb))
	t.Cleanup(func() {
		db.Exec("DELETE FROM rbac WHERE id = $1", rb.ID)
	})

	vars := map[string]string{"user": u.Username}
	uri := api.Router.GetRouteV2("GET", api.getUserPermissionHandler, vars)
	test.NotEmpty(t, uri)

	req := assets.NewAuthentifiedRequest(t, u, pass, "GET", uri, nil)
	w := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w, req)
	require.Equal(t, 200, w.Code)

	var summary sdk.PermissionSummary
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &summary))

	require.Empty(t, summary.Global)
	require.Len(t, summary.Projects, 1)
	proj, ok := summary.Projects[p.Key]
	require.True(t, ok, "project key %s should be in the summary", p.Key)
	require.Contains(t, proj.Roles, sdk.ProjectRoleRead)
}

func Test_getUserPermissionHandler_unknownUser(t *testing.T) {
	api, db, _ := newTestAPI(t)

	u, pass := assets.InsertAdminUser(t, db)

	vars := map[string]string{"user": "unknown-user-does-not-exist"}
	uri := api.Router.GetRouteV2("GET", api.getUserPermissionHandler, vars)
	test.NotEmpty(t, uri)

	req := assets.NewAuthentifiedRequest(t, u, pass, "GET", uri, nil)
	w := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w, req)
	require.Equal(t, 404, w.Code)
}
