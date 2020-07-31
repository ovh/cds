package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-gorp/gorp"
	"github.com/golang/mock/gomock"
	"github.com/ovh/cds/engine/api/repositoriesmanager"
	"github.com/ovh/cds/engine/api/services"
	"github.com/ovh/cds/engine/api/services/mock_services"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_WorkflowAsCodeRename(t *testing.T) {
	api, db, _ := newTestAPI(t)

	_, _ = assets.InsertService(t, db, t.Name()+"_HOOKS", sdk.TypeHooks)
	_, _ = assets.InsertService(t, db, t.Name()+"_VCS", sdk.TypeVCS)
	_, _ = assets.InsertService(t, db, t.Name()+"_REPO", sdk.TypeRepositories)

	// Setup a mock for all services called by the API
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	// The mock has been geenrated by mockgen: go get github.com/golang/mock/mockgen
	// If you have to regenerate thi mock you just have to run, from directory $GOPATH/src/github.com/ovh/cds/engine/api/services:
	// mockgen -source=http.go -destination=mock_services/services_mock.go Client
	servicesClients := mock_services.NewMockClient(ctrl)
	services.NewClient = func(_ gorp.SqlExecutor, _ []sdk.Service) services.Client {
		return servicesClients
	}
	defer func() {
		services.NewClient = services.NewDefaultClient
	}()

	// Create a project with a repository manager
	prjKey := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, api.Cache, prjKey, prjKey)
	u, pass := assets.InsertLambdaUser(t, db, &proj.ProjectGroups[0].Group)

	vcsServer := sdk.ProjectVCSServerLink{
		ProjectID: proj.ID,
		Name:      "github",
	}
	vcsServer.Set("token", "foo")
	vcsServer.Set("secret", "bar")
	assert.NoError(t, repositoriesmanager.InsertProjectVCSServerLink(context.TODO(), db, &vcsServer))

	UUID := sdk.UUID()

	servicesClients.EXPECT().
		DoJSONRequest(gomock.Any(), "POST", "/operations", gomock.Any(), gomock.Any()).
		DoAndReturn(
			func(ctx context.Context, method, path string, in interface{}, out interface{}) (http.Header, int, error) {
				ope := new(sdk.Operation)
				ope.UUID = UUID
				ope.Status = sdk.OperationStatusPending
				*(out.(*sdk.Operation)) = *ope
				return nil, 201, nil
			}).Times(2)

	servicesClients.EXPECT().
		DoJSONRequest(gomock.Any(), "GET", "/vcs/github/repos/fsamin/go-repo", gomock.Any(), gomock.Any(), gomock.Any()).MinTimes(0)

	servicesClients.EXPECT().
		DoJSONRequest(gomock.Any(), "GET", "/vcs/github/repos/fsamin/go-repo/branches", gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(
			func(ctx context.Context, method, path string, in interface{}, out interface{}, _ interface{}) (http.Header, int, error) {
				bs := []sdk.VCSBranch{}
				b := sdk.VCSBranch{
					DisplayID: "master",
				}
				bs = append(bs, b)
				out = bs
				return nil, 200, nil
			},
		).Times(4)

	servicesClients.EXPECT().
		DoJSONRequest(gomock.Any(), "GET", "/vcs/github/repos/fsamin/go-repo/branches/?branch=master", gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(
			func(ctx context.Context, method, path string, in interface{}, out interface{}, _ interface{}) (http.Header, int, error) {
				bs := []sdk.VCSBranch{}
				b := sdk.VCSBranch{
					DisplayID: "master",
				}
				bs = append(bs, b)
				out = bs
				return nil, 200, nil
			},
		).MaxTimes(3)

	gomock.InOrder(
		servicesClients.EXPECT().
			DoJSONRequest(gomock.Any(), "GET", gomock.Any(), gomock.Any(), gomock.Any()).
			DoAndReturn(
				func(ctx context.Context, method, path string, in interface{}, out interface{}) (http.Header, int, error) {
					ope := new(sdk.Operation)
					ope.URL = "https://github.com/fsamin/go-repo.git"
					ope.UUID = UUID
					ope.Status = sdk.OperationStatusDone
					ope.VCSServer = "github"
					ope.RepoFullName = "fsamin/go-repo"
					ope.RepositoryStrategy.Branch = "master"
					ope.Setup.Checkout.Branch = "master"
					ope.RepositoryInfo = new(sdk.OperationRepositoryInfo)
					ope.RepositoryInfo.Name = "fsamin/go-repo"
					ope.RepositoryInfo.DefaultBranch = "master"
					ope.RepositoryInfo.FetchURL = "https://github.com/fsamin/go-repo.git"
					ope.LoadFiles.Pattern = workflow.WorkflowAsCodePattern
					ope.LoadFiles.Results = map[string][]byte{
						"w-go-repo.yml": []byte(`
name: w-go-repo
version: v1.0
pipeline: build
application: go-repo
`),
						"go-repo.app.yml": []byte(`
name: go-repo
version: v1.0
repo: fsamin/go-repo
vcs_server: github
`),
						"go-repo.pip.yml": []byte(`name: build
version: v1.0`),
					}
					*(out.(*sdk.Operation)) = *ope
					return nil, 200, nil
				},
			).Times(2),
		servicesClients.EXPECT().
			DoJSONRequest(gomock.Any(), "GET", gomock.Any(), gomock.Any(), gomock.Any()).
			DoAndReturn(
				func(ctx context.Context, method, path string, in interface{}, out interface{}) (http.Header, int, error) {
					ope := new(sdk.Operation)
					ope.URL = "https://github.com/fsamin/go-repo.git"
					ope.UUID = UUID
					ope.Status = sdk.OperationStatusDone
					ope.VCSServer = "github"
					ope.RepoFullName = "fsamin/go-repo"
					ope.RepositoryStrategy.Branch = "master"
					ope.Setup.Checkout.Branch = "master"
					ope.RepositoryInfo = new(sdk.OperationRepositoryInfo)
					ope.RepositoryInfo.Name = "fsamin/go-repo"
					ope.RepositoryInfo.DefaultBranch = "master"
					ope.RepositoryInfo.FetchURL = "https://github.com/fsamin/go-repo.git"
					ope.LoadFiles.Pattern = workflow.WorkflowAsCodePattern
					ope.LoadFiles.Results = map[string][]byte{
						"w-go-repo.yml": []byte(`
name: w-go-repo-renamed
version: v1.0
pipeline: build
application: go-repo
`),
						"go-repo.app.yml": []byte(`
name: go-repo
version: v1.0
repo: fsamin/go-repo
vcs_server: github
`),
						"go-repo.pip.yml": []byte(`name: build
version: v1.0`),
					}
					*(out.(*sdk.Operation)) = *ope
					return nil, 200, nil
				},
			).Times(1),
	)

	gomock.InOrder(
		servicesClients.EXPECT().
			DoJSONRequest(gomock.Any(), "POST", "/task/bulk", gomock.Any(), gomock.Any()).
			DoAndReturn(
				func(ctx context.Context, method, path string, in interface{}, out interface{}) (http.Header, int, error) {
					actualHooks, ok := in.(map[string]sdk.NodeHook)
					require.True(t, ok)
					require.Len(t, actualHooks, 1)
					for k, h := range actualHooks {
						assert.Equal(t, "RepositoryWebHook", h.HookModelName)
						assert.Equal(t, "POST", h.Config["method"].Value)
						assert.Equal(t, "fsamin/go-repo", h.Config["repoFullName"].Value)
						assert.Equal(t, "github", h.Config["vcsServer"].Value)
						assert.Equal(t, "w-go-repo", h.Config["workflow"].Value)
						h.Config["webHookURL"] = sdk.WorkflowNodeHookConfigValue{
							Value:        "http://lolcat.host",
							Configurable: false,
						}
						actualHooks[k] = h
					}
					out = actualHooks
					return nil, 200, nil
				},
			),
		servicesClients.EXPECT().
			DoJSONRequest(gomock.Any(), "POST", "/task/bulk", gomock.Any(), gomock.Any()).
			DoAndReturn(
				func(ctx context.Context, method, path string, in interface{}, out interface{}) (http.Header, int, error) {
					actualHooks, ok := in.(map[string]sdk.NodeHook)
					require.True(t, ok)
					require.Len(t, actualHooks, 1)
					for k, h := range actualHooks {
						assert.Equal(t, "RepositoryWebHook", h.HookModelName)
						assert.Equal(t, "POST", h.Config["method"].Value)
						assert.Equal(t, "fsamin/go-repo", h.Config["repoFullName"].Value)
						assert.Equal(t, "github", h.Config["vcsServer"].Value)
						assert.Equal(t, "w-go-repo-renamed", h.Config["workflow"].Value)
						h.Config["webHookURL"] = sdk.WorkflowNodeHookConfigValue{
							Value:        "http://lolcat.host",
							Configurable: false,
						}
						actualHooks[k] = h
					}
					out = actualHooks
					return nil, 200, nil
				},
			),
	)

	servicesClients.EXPECT().
		DoJSONRequest(gomock.Any(), "GET", "/vcs/github/webhooks", gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(
			func(ctx context.Context, method, path string, in interface{}, out interface{}, _ interface{}) (http.Header, int, error) {
				*(out.(*repositoriesmanager.WebhooksInfos)) = repositoriesmanager.WebhooksInfos{
					WebhooksSupported: true,
					WebhooksDisabled:  false,
					Icon:              sdk.GitHubIcon,
					Events: []string{
						"push",
					},
				}
				return nil, 200, nil
			},
		).Times(2)

	servicesClients.EXPECT().
		DoJSONRequest(gomock.Any(), "POST", "/vcs/github/repos/fsamin/go-repo/grant", gomock.Any(), gomock.Any(), gomock.Any())

	servicesClients.EXPECT().
		DoJSONRequest(gomock.Any(), "POST", "/vcs/github/repos/fsamin/go-repo/hooks", gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(
			func(ctx context.Context, method, path string, in interface{}, out interface{}, _ interface{}) (http.Header, int, error) {
				vcsHooks := in.(*sdk.VCSHook)

				require.Len(t, vcsHooks.Events, 0, "events list should be empty, default value is set by vcs")
				vcsHooks.Events = []string{"push"}

				vcsHooks.ID = sdk.UUID()
				*(out.(*sdk.VCSHook)) = *vcsHooks
				return nil, 200, nil
			},
		).Times(1)

	servicesClients.EXPECT().
		DoJSONRequest(gomock.Any(), "PUT", "/vcs/github/repos/fsamin/go-repo/hooks", gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(
			func(ctx context.Context, method, path string, in interface{}, out interface{}, _ interface{}) (http.Header, int, error) {
				vcsHooks := in.(*sdk.VCSHook)
				vcsHooks.ID = sdk.UUID()
				require.Len(t, vcsHooks.Events, 0, "events list should be empty, default value is set by vcs")

				vcsHooks.Events = []string{
					"push",
				}
				*(out.(*sdk.VCSHook)) = *vcsHooks
				return nil, 200, nil
			},
		).Times(1)

	// Perform a "import as-code operation" to create a new workflow
	ope := `{"repo_fullname":"fsamin/go-repo",  "vcs_server": "github", "url":"https://github.com/fsamin/go-repo.git","strategy":{"connection_type":"https","ssh_key":"","user":"","password":"","branch":"","default_branch":"master","pgp_key":""},"setup":{"checkout":{"branch":"master"}}}`
	uri := api.Router.GetRoute("POST", api.postImportAsCodeHandler, map[string]string{
		"permProjectKey": proj.Key,
	})
	req, err := http.NewRequest("POST", uri, strings.NewReader(ope))
	require.NoError(t, err)
	assets.AuthentifyRequest(t, req, u, pass)
	w := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w, req)
	require.Equal(t, 201, w.Code)
	// Fake wait api event for operation done
	time.Sleep(time.Second)

	uri = api.Router.GetRoute("POST", api.postPerformImportAsCodeHandler, map[string]string{
		"permProjectKey": prjKey,
		"uuid":           UUID,
	})
	req, err = http.NewRequest("POST", uri, nil)
	require.NoError(t, err)
	assets.AuthentifyRequest(t, req, u, pass)
	w = httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w, req)
	require.Equal(t, 200, w.Code)
	t.Logf(w.Body.String())

	wk, err := workflow.Load(context.Background(), db, api.Cache, *proj, "w-go-repo", workflow.LoadOptions{})
	require.NoError(t, err)
	require.Len(t, wk.WorkflowData.GetHooks(), 1)

	for _, h := range wk.WorkflowData.GetHooks() {
		log.Debug("--> %T %+v", h, h)
		require.Equal(t, "RepositoryWebHook", h.HookModelName)
		require.Equal(t, "push", h.Config["eventFilter"].Value)
		require.Equal(t, "Github", h.Config["hookIcon"].Value)
		require.Equal(t, "POST", h.Config["method"].Value)
		require.Equal(t, proj.Key, h.Config["project"].Value)
		require.Equal(t, "fsamin/go-repo", h.Config["repoFullName"].Value)
		require.Equal(t, "github", h.Config["vcsServer"].Value)
		require.Equal(t, wk.Name, h.Config["workflow"].Value)
	}

	// Then we will trigger a run of the workflow wich should trigger an as-code operation with a renamed workflow
	vars := map[string]string{
		"key":              proj.Key,
		"permWorkflowName": wk.Name,
	}
	uri = api.Router.GetRoute("POST", api.postWorkflowRunHandler, vars)
	require.NotEmpty(t, uri)

	opts := &sdk.WorkflowRunPostHandlerOption{
		Hook: &sdk.WorkflowNodeRunHookEvent{
			WorkflowNodeHookUUID: wk.WorkflowData.Node.Hooks[0].UUID,
			Payload: map[string]string{
				"git.branch": "master",
			},
		},
	}
	req = assets.NewAuthentifiedRequest(t, u, pass, "POST", uri, opts)

	//Do the request
	rec := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	assert.Equal(t, 202, rec.Code)

	if rec.Code != 202 {
		t.Logf("body => %s", rec.Body.String())
		t.FailNow()
	}

	var wrun sdk.WorkflowRun
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &wrun))

	require.NoError(t, waitCraftinWorkflow(t, api, api.mustDB(), wrun.ID))
	wr, _ := workflow.LoadRunByID(db, wrun.ID, workflow.LoadRunOptions{})

	assert.NotEqual(t, "Fail", wr.Status)

	wk, err = workflow.Load(context.Background(), db, api.Cache, *proj, "w-go-repo-renamed", workflow.LoadOptions{})
	assert.NoError(t, err)
	assert.NotNil(t, wk)

	_, err = workflow.Load(context.Background(), db, api.Cache, *proj, "w-go-repo", workflow.LoadOptions{})
	assert.Error(t, err)

	require.Len(t, wk.WorkflowData.GetHooks(), 1)

	for _, h := range wk.WorkflowData.GetHooks() {
		log.Debug("--> %T %+v", h, h)
		require.Equal(t, "RepositoryWebHook", h.HookModelName)
		require.Equal(t, "push", h.Config["eventFilter"].Value)
		require.Equal(t, "Github", h.Config["hookIcon"].Value)
		require.Equal(t, "POST", h.Config["method"].Value)
		require.Equal(t, proj.Key, h.Config["project"].Value)
		require.Equal(t, "fsamin/go-repo", h.Config["repoFullName"].Value)
		require.Equal(t, "github", h.Config["vcsServer"].Value)
		require.Equal(t, wk.Name, h.Config["workflow"].Value)
	}
}
