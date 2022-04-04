package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http/httptest"
	"strings"
	"testing"

	yaml "github.com/ghodss/yaml"
	"github.com/ovh/cds/engine/api/rbac"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/sdk"
	"github.com/stretchr/testify/require"
)

func Test_crudVCSOnProjectLambdaUserForbidden(t *testing.T) {
	api, db, _ := newTestAPI(t)

	u, pass := assets.InsertLambdaUser(t, db)
	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))

	vars := map[string]string{
		"projectKey": proj.Key,
	}
	uriPost := api.Router.GetRouteV2("POST", api.postVCSProjectHandler, vars)
	test.NotEmpty(t, uriPost)

	req := assets.NewAuthentifiedRequest(t, u, pass, "POST", uriPost, nil)

	w := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w, req)
	require.Equal(t, 403, w.Code)

	uriGet := api.Router.GetRouteV2("GET", api.getVCSProjectAllHandler, vars)
	test.NotEmpty(t, uriGet)

	reqGet := assets.NewAuthentifiedRequest(t, u, pass, "GET", uriGet, nil)
	w2 := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w2, reqGet)
	require.Equal(t, 403, w2.Code)
}

func Test_crudVCSOnProjectLambdaUserOK(t *testing.T) {
	api, db, _ := newTestAPI(t)

	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))
	user1, pass := assets.InsertLambdaUser(t, db)

	perm := fmt.Sprintf(`name: perm-test-%s
projects:
  - role: manage
    projects: [%s]
    users: [%s]
`, proj.Key, proj.Key, user1.Username)

	var rb sdk.RBAC
	require.NoError(t, yaml.Unmarshal([]byte(perm), &rb))

	rb.Projects[0].RBACProjectKeys = []string{proj.Key}
	rb.Projects[0].RBACUsersIDs = []string{user1.ID}

	require.NoError(t, rbac.Insert(context.Background(), db, &rb))

	vars := map[string]string{
		"projectKey": proj.Key,
	}
	uri := api.Router.GetRouteV2("POST", api.postVCSProjectHandler, vars)
	test.NotEmpty(t, uri)
	req := assets.NewAuthentifiedRequest(t, user1, pass, "POST", uri, nil)

	body := `version: v1.0
name: my_vcs_server
type: bitbucketserver
description: "it's the test vcs server on project"
url: "http://my-vcs-server.localhost"
auth:
    user: the-username
    password: the-password
`

	// Here, we insert the vcs server as a CDS user (not administrator)
	req.Body = io.NopCloser(strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-yaml")

	w := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w, req)
	require.Equal(t, 201, w.Code)

	// Then, get the vcs server
	uriGet := api.Router.GetRouteV2("GET", api.getVCSProjectAllHandler, vars)
	test.NotEmpty(t, uriGet)

	reqGet := assets.NewAuthentifiedRequest(t, user1, pass, "GET", uriGet, nil)
	w2 := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w2, reqGet)
	require.Equal(t, 200, w2.Code)

	vcsProjects := []sdk.VCSProject{}
	require.NoError(t, json.Unmarshal(w2.Body.Bytes(), &vcsProjects))
	require.Len(t, vcsProjects, 1)
}

func Test_crudVCSOnProjectAdminOk(t *testing.T) {
	api, db, _ := newTestAPI(t)

	u, pass := assets.InsertAdminUser(t, db)
	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))

	vars := map[string]string{
		"projectKey": proj.Key,
	}
	uri := api.Router.GetRouteV2("POST", api.postVCSProjectHandler, vars)
	test.NotEmpty(t, uri)
	req := assets.NewAuthentifiedRequest(t, u, pass, "POST", uri, nil)

	body := `version: v1.0
name: my_vcs_server
type: bitbucketserver
description: "it's the test vcs server on project"
url: "http://my-vcs-server.localhost"
auth:
    user: the-username
    password: the-password
`

	// Here, we insert the vcs server as a CDS administrator
	req.Body = io.NopCloser(strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-yaml")

	w := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w, req)
	require.Equal(t, 201, w.Code)

	// Then, get the vcs server in the list of vcs
	uriGetAll := api.Router.GetRouteV2("GET", api.getVCSProjectAllHandler, vars)
	test.NotEmpty(t, uriGetAll)

	reqGetAll := assets.NewAuthentifiedRequest(t, u, pass, "GET", uriGetAll, nil)
	w2 := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w2, reqGetAll)
	require.Equal(t, 200, w2.Code)

	vcsProjects := []sdk.VCSProject{}
	require.NoError(t, json.Unmarshal(w2.Body.Bytes(), &vcsProjects))
	require.Len(t, vcsProjects, 1)

	// Then, try to get the vcs server directly
	vars["vcsProjectName"] = "my_vcs_server"
	uriGet := api.Router.GetRouteV2("GET", api.getVCSProjectHandler, vars)
	test.NotEmpty(t, uriGet)

	reqGet := assets.NewAuthentifiedRequest(t, u, pass, "GET", uriGet, nil)
	w3 := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w3, reqGet)
	require.Equal(t, 200, w3.Code)

	vcsProject := sdk.VCSProject{}
	require.NoError(t, json.Unmarshal(w3.Body.Bytes(), &vcsProject))
	require.Equal(t, "my_vcs_server", vcsProject.Name)
	require.Empty(t, vcsProject.Auth)

	// delete the vcs project
	uriDelete := api.Router.GetRouteV2("DELETE", api.deleteVCSProjectHandler, vars)
	test.NotEmpty(t, uriDelete)

	reqDelete := assets.NewAuthentifiedRequest(t, u, pass, "DELETE", uriDelete, nil)
	w4 := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w4, reqDelete)
	require.Equal(t, 204, w4.Code)

	reqGetAll2 := assets.NewAuthentifiedRequest(t, u, pass, "GET", uriGetAll, nil)
	w5 := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w5, reqGetAll2)
	require.Equal(t, 200, w5.Code)

	vcsProjects2 := []sdk.VCSProject{}
	require.NoError(t, json.Unmarshal(w5.Body.Bytes(), &vcsProjects2))
	require.Len(t, vcsProjects2, 0)
}
