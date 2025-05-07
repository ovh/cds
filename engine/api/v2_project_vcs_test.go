package api

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ovh/cds/engine/api/keys"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/vcs"
	"github.com/ovh/cds/engine/gorpmapper"

	"github.com/ovh/cds/engine/api/services"
	"github.com/ovh/cds/engine/api/services/mock_services"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/sdk"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
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

	newKey, err := keys.GenerateSSHKey("mykey")
	require.NoError(t, err)
	k := sdk.ProjectKey{
		Private:   newKey.Private,
		Public:    newKey.Public,
		KeyID:     newKey.KeyID,
		ProjectID: proj.ID,
		Name:      "mykey",
		Type:      sdk.KeyTypeSSH,
	}
	require.NoError(t, project.InsertKey(db, &k))

	assets.InsertRBAcProject(t, db, "manage", proj.Key, *user1)
	assets.InsertRBAcProject(t, db, "read", proj.Key, *user1)

	// Mock VCS
	s, _ := assets.InsertService(t, db, t.Name()+"_VCS", sdk.TypeVCS)
	// Setup a mock for all services called by the API
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	servicesClients := mock_services.NewMockClient(ctrl)
	services.NewClient = func(_ []sdk.Service) services.Client {
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
  username: the-username
  token: the-password
  sshKeyName: mykey
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

	newKey, err := keys.GenerateSSHKey("mykey")
	require.NoError(t, err)
	k := sdk.ProjectKey{
		Private:   newKey.Private,
		Public:    newKey.Public,
		KeyID:     newKey.KeyID,
		ProjectID: proj.ID,
		Name:      "mykey",
		Type:      sdk.KeyTypeSSH,
	}
	require.NoError(t, project.InsertKey(db, &k))

	gpgKey, err := keys.GeneratePGPKeyPair("my-gpg-key", "my-gpg-key", "my-gpg-key@localhost.local")
	require.NoError(t, err)
	k2 := sdk.ProjectKey{
		Private:   gpgKey.Private,
		Public:    gpgKey.Public,
		KeyID:     gpgKey.KeyID,
		ProjectID: proj.ID,
		Name:      gpgKey.Name,
		Type:      sdk.KeyTypePGP,
		LongKeyID: gpgKey.LongKeyID,
	}
	require.NoError(t, project.InsertKey(db, &k2))

	// Mock VCS
	s, _ := assets.InsertService(t, db, t.Name()+"_VCS", sdk.TypeVCS)
	// Setup a mock for all services called by the API
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	servicesClients := mock_services.NewMockClient(ctrl)
	services.NewClient = func(_ []sdk.Service) services.Client {
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
  username: the-username
  token: the-password
  sshKeyName: mykey
  gpgKeyName: my-gpg-key
  emailAddress: my-gpg-key@localhost.local
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

	vcsProjectFromDB, err := vcs.LoadVCSByProject(context.TODO(), db, proj.Key, "my_vcs_server", gorpmapper.GetOptions.WithDecryption)
	require.NoError(t, err)

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
	require.Empty(t, vcsProject.Auth.SSHPrivateKey)
	require.Empty(t, vcsProject.Auth.Token)

	// Try to get key used in the VCS
	urlGetKey := api.Router.GetRouteV2("GET", api.GetVCSPGKeyHandler, map[string]string{
		"gpgKeyID": k2.LongKeyID,
	})
	reqGetKey := assets.NewAuthentifiedRequest(t, u, pass, "GET", urlGetKey, nil)
	rec := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, reqGetKey)
	require.Equal(t, 200, rec.Code)
	VCSUserGPGKeys := []sdk.VCSUserGPGKey{}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &VCSUserGPGKeys))
	require.Len(t, VCSUserGPGKeys, 1)
	require.Equal(t, proj.Key, VCSUserGPGKeys[0].ProjectKey)
	require.Equal(t, vcsProjectFromDB.Name, VCSUserGPGKeys[0].VCSProjectName)
	require.Equal(t, vcsProjectFromDB.Auth.Username, VCSUserGPGKeys[0].Username)
	require.Equal(t, gpgKey.LongKeyID, VCSUserGPGKeys[0].KeyID)
	require.Equal(t, vcsProjectFromDB.Auth.GPGKeyName, VCSUserGPGKeys[0].KeyName)
	require.Equal(t, gpgKey.Public, VCSUserGPGKeys[0].PublicKey)

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

func Test_crudVCSOnPublicProject(t *testing.T) {
	api, db, _ := newTestAPI(t)

	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))
	user1, pass := assets.InsertLambdaUser(t, db)

	newKey, err := keys.GenerateSSHKey("mykey")
	require.NoError(t, err)
	k := sdk.ProjectKey{
		Private:   newKey.Private,
		Public:    newKey.Public,
		KeyID:     newKey.KeyID,
		ProjectID: proj.ID,
		Name:      "mykey",
		Type:      sdk.KeyTypeSSH,
	}
	require.NoError(t, project.InsertKey(db, &k))

	assets.InsertRBAcProject(t, db, "manage", proj.Key, *user1)
	assets.InsertRBAcPublicProject(t, db, "read", proj.Key)

	// Mock VCS
	s, _ := assets.InsertService(t, db, t.Name()+"_VCS", sdk.TypeVCS)
	// Setup a mock for all services called by the API
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	servicesClients := mock_services.NewMockClient(ctrl)
	services.NewClient = func(_ []sdk.Service) services.Client {
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
  username: the-username
  token: the-password
  sshKeyName: mykey
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
