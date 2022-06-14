package api

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-gorp/gorp"
	"github.com/golang/mock/gomock"
	"github.com/ovh/cds/engine/api/services"
	"github.com/ovh/cds/engine/api/services/mock_services"
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

	assets.InsertRBAcProject(t, db, "manage", proj.Key, *user1)
	assets.InsertRBAcProject(t, db, "read", proj.Key, *user1)

	// Mock VCS
	s, _ := assets.InsertService(t, db, t.Name()+"_VCS", sdk.TypeVCS)
	// Setup a mock for all services called by the API
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	servicesClients := mock_services.NewMockClient(ctrl)
	services.NewClient = func(_ gorp.SqlExecutor, _ []sdk.Service) services.Client {
		return servicesClients
	}
	defer func() {
		_ = services.Delete(db, s)
		services.NewClient = services.NewDefaultClient
	}()

	servicesClients.EXPECT().
		DoJSONRequest(gomock.Any(), "GET", "/vcs/my_vcs_server/repos", gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(
			func(ctx context.Context, method, path string, in interface{}, out interface{}, _ interface{}) (http.Header, int, error) {
				repos := []sdk.VCSRepo{}
				out = repos
				return nil, 200, nil
			},
		).MaxTimes(1)

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

	// Mock VCS
	s, _ := assets.InsertService(t, db, t.Name()+"_VCS", sdk.TypeVCS)
	// Setup a mock for all services called by the API
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	servicesClients := mock_services.NewMockClient(ctrl)
	services.NewClient = func(_ gorp.SqlExecutor, _ []sdk.Service) services.Client {
		return servicesClients
	}
	defer func() {
		_ = services.Delete(db, s)
		services.NewClient = services.NewDefaultClient
	}()

	servicesClients.EXPECT().
		DoJSONRequest(gomock.Any(), "GET", "/vcs/my_vcs_server/repos", gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(
			func(ctx context.Context, method, path string, in interface{}, out interface{}, _ interface{}) (http.Header, int, error) {
				repos := []sdk.VCSRepo{}
				out = repos
				return nil, 200, nil
			},
		).MaxTimes(1)

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
	vars["vcsIdentifier"] = "my_vcs_server"
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
