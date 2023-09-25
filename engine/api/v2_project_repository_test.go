package api

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/go-gorp/gorp"
	"github.com/golang/mock/gomock"
	"github.com/ovh/cds/engine/api/services"
	"github.com/ovh/cds/engine/api/services/mock_services"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/sdk"
	"github.com/stretchr/testify/require"
)

func Test_crudRepositoryOnProjectLambdaUserOK(t *testing.T) {
	api, db, _ := newTestAPI(t)

	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))
	user1, pass := assets.InsertLambdaUser(t, db)

	vcsProj := assets.InsertTestVCSProject(t, db, proj.ID, "vcs-github", "github")

	// Insert rbac
	assets.InsertRBAcProject(t, db, "manage", proj.Key, *user1)
	assets.InsertRBAcProject(t, db, "read", proj.Key, *user1)

	// Mock VCS
	s, _ := assets.InsertService(t, db, t.Name()+"_VCS", sdk.TypeVCS)
	sHooks, _ := assets.InsertService(t, db, t.Name()+"_HOOK", sdk.TypeHooks)
	// Setup a mock for all services called by the API
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	servicesClients := mock_services.NewMockClient(ctrl)
	services.NewClient = func(_ gorp.SqlExecutor, _ []sdk.Service) services.Client {
		return servicesClients
	}
	defer func() {
		_ = services.Delete(db, s)
		_ = services.Delete(db, sHooks)
		services.NewClient = services.NewDefaultClient
	}()

	servicesClients.EXPECT().
		DoJSONRequest(gomock.Any(), "GET", "/vcs/vcs-github/repos/ovh/cds", gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(
			func(ctx context.Context, method, path string, in interface{}, out interface{}, _ interface{}) (http.Header, int, error) {
				r := sdk.VCSRepo{
					Name:         "ovh/cds",
					HTTPCloneURL: "http://fakeURL",
				}
				*(out.(*sdk.VCSRepo)) = r
				return nil, 200, nil
			},
		).MaxTimes(1)
	servicesClients.EXPECT().DoJSONRequest(gomock.Any(), "GET", "/vcs/vcs-github/repos/ovh/cds/branches/?branch=&default=true", gomock.Any(), gomock.Any(), gomock.Any()).Times(1)

	// Creation request
	repo := sdk.ProjectRepository{
		Name:       "ovh/cds",
		ProjectKey: proj.Key,
	}

	vars := map[string]string{
		"projectKey":    proj.Key,
		"vcsIdentifier": vcsProj.ID,
	}
	uri := api.Router.GetRouteV2("POST", api.postProjectRepositoryHandler, vars)
	test.NotEmpty(t, uri)
	req := assets.NewAuthentifiedRequest(t, user1, pass, "POST", uri, nil)

	bts, _ := json.Marshal(repo)
	// Here, we insert the vcs server as a CDS user (not administrator)
	req.Body = io.NopCloser(bytes.NewReader(bts))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w, req)
	require.Equal(t, 201, w.Code)

	// Then, get the vcs server
	uriGet := api.Router.GetRouteV2("GET", api.getVCSProjectRepositoryAllHandler, vars)
	test.NotEmpty(t, uriGet)
	reqGet := assets.NewAuthentifiedRequest(t, user1, pass, "GET", uriGet, nil)
	w2 := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w2, reqGet)
	require.Equal(t, 200, w2.Code)

	var repositories []sdk.ProjectRepository
	require.NoError(t, json.Unmarshal(w2.Body.Bytes(), &repositories))
	require.Len(t, repositories, 1)

	servicesClients.EXPECT().DoJSONRequest(gomock.Any(), "DELETE", "/task/"+repositories[0].ID, gomock.Any(), gomock.Any(), gomock.Any()).Times(1)

	// Then Delete repository
	varsDelete := vars
	varsDelete["repositoryIdentifier"] = url.PathEscape("ovh/cds")
	uriDelete := api.Router.GetRouteV2("DELETE", api.deleteProjectRepositoryHandler, varsDelete)
	test.NotEmpty(t, uriDelete)
	reqDelete := assets.NewAuthentifiedRequest(t, user1, pass, "DELETE", uriDelete, nil)
	w3 := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w3, reqDelete)
	require.Equal(t, 200, w3.Code)

	// Then check if repository has been deleted
	w4 := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w4, reqGet)
	require.Equal(t, 200, w4.Code)
	require.NoError(t, json.Unmarshal(w4.Body.Bytes(), &repositories))
	require.Len(t, repositories, 0)
}
