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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ovh/cds/engine/api/repositoriesmanager"
	"github.com/ovh/cds/engine/api/services"
	"github.com/ovh/cds/engine/api/services/mock_services"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func Test_WorkflowAsCodeWithNoHook_ShouldGive_AnAutomaticRepoWebHook(t *testing.T) {
	api, db, _ := newTestAPI(t)

	_, _ = assets.InsertService(t, db, t.Name()+"_HOOKS", sdk.TypeHooks)
	_, _ = assets.InsertService(t, db, t.Name()+"_VCS", sdk.TypeVCS)
	_, _ = assets.InsertService(t, db, t.Name()+"_REPO", sdk.TypeRepositories)

	// Setup a mock for all services called by the API
	ctrl := gomock.NewController(t)
	t.Cleanup(func() { ctrl.Finish() })
	// The mock has been geenrated by mockgen: go get github.com/golang/mock/mockgen
	// If you have to regenerate thi mock you just have to run, from directory $GOPATH/src/github.com/ovh/cds/engine/api/services:
	// mockgen -source=http.go -destination=mock_services/services_mock.go Client
	servicesClients := mock_services.NewMockClient(ctrl)
	services.NewClient = func(_ gorp.SqlExecutor, _ []sdk.Service) services.Client { return servicesClients }
	t.Cleanup(func() { services.NewClient = services.NewDefaultClient })

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
			})

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
		).MaxTimes(3)

	servicesClients.EXPECT().
		DoJSONRequest(gomock.Any(), "GET", "/operations/"+UUID, gomock.Any(), gomock.Any()).
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
		).Times(2)

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
		)

	servicesClients.EXPECT().
		DoJSONRequest(gomock.Any(), "POST", "/vcs/github/repos/fsamin/go-repo/grant", gomock.Any(), gomock.Any(), gomock.Any())

	servicesClients.EXPECT().
		DoJSONRequest(gomock.Any(), "POST", "/vcs/github/repos/fsamin/go-repo/hooks", gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(
			func(ctx context.Context, method, path string, in interface{}, out interface{}, _ interface{}) (http.Header, int, error) {
				log.Debug("--> %T %+v", in, in)

				vcsHooks, ok := in.(*sdk.VCSHook)
				require.True(t, ok)

				require.Len(t, vcsHooks.Events, 0, "events list should be empty, default value is set by vcs")
				vcsHooks.Events = []string{"push"}

				assert.Equal(t, "POST", vcsHooks.Method)
				assert.Equal(t, "http://lolcat.host", vcsHooks.URL)
				vcsHooks.ID = sdk.UUID()
				*(out.(*sdk.VCSHook)) = *vcsHooks

				return nil, 200, nil
			},
		)

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
	assert.Equal(t, 201, w.Code)
	// Fake wait api event for operation done
	time.Sleep(time.Second)

	uri = api.Router.GetRoute("POST", api.postPerformImportAsCodeHandler, map[string]string{
		"permProjectKey": prjKey,
		"uuid":           UUID,
	})
	req, err = http.NewRequest("POST", uri, nil)
	require.NoError(t, err)
	assets.AuthentifyRequest(t, req, u, pass)

	// Do the request
	w = httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)
	t.Logf(w.Body.String())

	wk, err := workflow.Load(context.Background(), db, api.Cache, *proj, "w-go-repo", workflow.LoadOptions{})
	assert.NoError(t, err)
	assert.NotNil(t, wk)

	require.Len(t, wk.WorkflowData.GetHooks(), 1)

	for _, h := range wk.WorkflowData.GetHooks() {
		log.Debug("--> %T %+v", h, h)
		assert.Equal(t, "RepositoryWebHook", h.HookModelName)
		assert.Equal(t, "push", h.Config["eventFilter"].Value)
		assert.Equal(t, "Github", h.Config["hookIcon"].Value)
		assert.Equal(t, "POST", h.Config["method"].Value)
		assert.Equal(t, proj.Key, h.Config["project"].Value)
		assert.Equal(t, "fsamin/go-repo", h.Config["repoFullName"].Value)
		assert.Equal(t, "github", h.Config["vcsServer"].Value)
		assert.Equal(t, wk.Name, h.Config["workflow"].Value)
		assert.NotEmpty(t, h.Config["webHookID"].Value)
	}
}

func Test_WorkflowAsCodeWithDefaultHook_ShouldGive_TheSameRepoWebHook(t *testing.T) {
	// We create a new workflow from a repository in which there is a .cds directory
	// We check that all the as-code operation is successfull and create the right hook
	// Then we trigger the workflow, this aims to run another "as-code operation"
	// This should not change the hooks

	api, db, _ := newTestAPI(t)

	_, _ = assets.InsertService(t, db, t.Name()+"_HOOKS", sdk.TypeHooks)
	_, _ = assets.InsertService(t, db, t.Name()+"_VCS", sdk.TypeVCS)
	_, _ = assets.InsertService(t, db, t.Name()+"_REPO", sdk.TypeRepositories)

	// Setup a mock for all services called by the API
	ctrl := gomock.NewController(t)
	t.Cleanup(func() { ctrl.Finish() })
	// The mock has been geenrated by mockgen: go get github.com/golang/mock/mockgen
	// If you have to regenerate thi mock you just have to run, from directory $GOPATH/src/github.com/ovh/cds/engine/api/services:
	// mockgen -source=http.go -destination=mock_services/services_mock.go Client
	servicesClients := mock_services.NewMockClient(ctrl)
	services.NewClient = func(_ gorp.SqlExecutor, _ []sdk.Service) services.Client { return servicesClients }
	t.Cleanup(func() { services.NewClient = services.NewDefaultClient })

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
	require.NoError(t, repositoriesmanager.InsertProjectVCSServerLink(context.TODO(), db, &vcsServer))

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

	servicesClients.EXPECT().
		DoJSONRequest(gomock.Any(), "GET", "/operations/"+UUID, gomock.Any(), gomock.Any()).
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
pipeline_hooks:
- type: RepositoryWebHook
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
		).Times(3)

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
		).Times(1) // This must be called only once

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
		).Times(1)

	servicesClients.EXPECT().
		DoJSONRequest(gomock.Any(), "POST", "/vcs/github/repos/fsamin/go-repo/grant", gomock.Any(), gomock.Any(), gomock.Any()).Times(1)

	servicesClients.EXPECT().
		DoJSONRequest(gomock.Any(), "POST", "/vcs/github/repos/fsamin/go-repo/hooks", gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(
			func(ctx context.Context, method, path string, in interface{}, out interface{}, _ interface{}) (http.Header, int, error) {
				log.Debug("--> %T %+v", in, in)

				vcsHooks, ok := in.(*sdk.VCSHook)
				require.True(t, ok)

				require.Len(t, vcsHooks.Events, 0, "events list should be empty, default value is set by vcs")
				vcsHooks.Events = []string{"push"}

				assert.Equal(t, "POST", vcsHooks.Method)
				assert.Equal(t, "http://lolcat.host", vcsHooks.URL)

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

	// Do the request
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

	// Then we will trigger a run of the workflow wich should trigger an as-code operation
	uri = api.Router.GetRoute("POST", api.postWorkflowRunHandler, map[string]string{
		"key":              proj.Key,
		"permWorkflowName": wk.Name,
	})
	require.NotEmpty(t, uri)
	req = assets.NewAuthentifiedRequest(t, u, pass, "POST", uri, &sdk.WorkflowRunPostHandlerOption{
		Hook: &sdk.WorkflowNodeRunHookEvent{
			WorkflowNodeHookUUID: wk.WorkflowData.Node.Hooks[0].UUID,
			Payload: map[string]string{
				"git.branch": "master",
			},
		},
	})
	rec := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	if !assert.Equal(t, 202, rec.Code) {
		t.Logf("body => %s", rec.Body.String())
		t.FailNow()
	}
	var wrun sdk.WorkflowRun
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &wrun))

	require.NoError(t, waitCraftinWorkflow(t, api, api.mustDB(), wrun.ID))
	wr, err := workflow.LoadRunByID(db, wrun.ID, workflow.LoadRunOptions{})
	require.NoError(t, nil)
	require.NotEqual(t, "Fail", wr.Status)

	wk, err = workflow.Load(context.Background(), db, api.Cache, *proj, "w-go-repo", workflow.LoadOptions{})
	require.NoError(t, err)
	require.NotNil(t, wk)
	require.Len(t, wk.WorkflowData.GetHooks(), 1)

	for _, h := range wk.WorkflowData.GetHooks() {
		log.Debug("--> %T %+v", h, h)
		assert.Equal(t, "RepositoryWebHook", h.HookModelName)
		assert.Equal(t, "push", h.Config["eventFilter"].Value)
		assert.Equal(t, "Github", h.Config["hookIcon"].Value)
		assert.Equal(t, "POST", h.Config["method"].Value)
		assert.Equal(t, proj.Key, h.Config["project"].Value)
		assert.Equal(t, "fsamin/go-repo", h.Config["repoFullName"].Value)
		assert.Equal(t, "github", h.Config["vcsServer"].Value)
		assert.Equal(t, wk.Name, h.Config["workflow"].Value)
	}
}

func Test_WorkflowAsCodeWithDefaultHookAndAScheduler_ShouldGive_TheSameRepoWebHookAndTheScheduler(t *testing.T) {
	// We create a new workflow from a repository in which there is a .cds directory
	// We check that all the as-code operation is successfull and create the right hook
	// Then we trigger the workflow, this aims to run another "as-code operation"
	// This should not change the hooks

	api, db, _ := newTestAPI(t)

	_, _ = assets.InsertService(t, db, t.Name()+"_HOOKS", sdk.TypeHooks)
	_, _ = assets.InsertService(t, db, t.Name()+"_VCS", sdk.TypeVCS)
	_, _ = assets.InsertService(t, db, t.Name()+"_REPO", sdk.TypeRepositories)

	// Setup a mock for all services called by the API
	ctrl := gomock.NewController(t)
	t.Cleanup(func() { ctrl.Finish() })
	// The mock has been geenrated by mockgen: go get github.com/golang/mock/mockgen
	// If you have to regenerate thi mock you just have to run, from directory $GOPATH/src/github.com/ovh/cds/engine/api/services:
	// mockgen -source=http.go -destination=mock_services/services_mock.go Client
	servicesClients := mock_services.NewMockClient(ctrl)
	services.NewClient = func(_ gorp.SqlExecutor, _ []sdk.Service) services.Client { return servicesClients }
	t.Cleanup(func() { services.NewClient = services.NewDefaultClient })

	// Create a project with a repository manager
	prjKey := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, api.Cache, prjKey, prjKey)
	u, jwt := assets.InsertLambdaUser(t, db, &proj.ProjectGroups[0].Group)

	vcsServer := sdk.ProjectVCSServerLink{
		ProjectID: proj.ID,
		Name:      "github",
	}
	vcsServer.Set("token", "foo")
	vcsServer.Set("secret", "bar")
	assert.NoError(t, repositoriesmanager.InsertProjectVCSServerLink(context.TODO(), db, &vcsServer))

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

	operationUUID := sdk.UUID()

	servicesClients.EXPECT().
		DoJSONRequest(gomock.Any(), "POST", "/operations", gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, method, path string, in interface{}, out interface{}) (http.Header, int, error) {
			ope := new(sdk.Operation)
			ope.UUID = operationUUID
			*(out.(*sdk.Operation)) = *ope
			return nil, 200, nil
		}).Times(2)

	servicesClients.EXPECT().
		DoJSONRequest(gomock.Any(), "GET", "/operations/"+operationUUID, gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(
			func(ctx context.Context, method, path string, in interface{}, out interface{}) (http.Header, int, error) {
				ope := new(sdk.Operation)
				ope.URL = "https://github.com/fsamin/go-repo.git"
				ope.UUID = operationUUID
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
pipeline_hooks:
- type: RepositoryWebHook
- type: Scheduler
  config:
    cron: 0/15 * * * *
    payload: |-
      {
        "git.author": "",
        "git.branch": "master",
        "git.hash": "",
        "git.hash.before": "",
        "git.message": "",
        "git.repository": "fsamin/go-repo",
        "git.tag": ""
      }
    timezone: UTC
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
		).Times(3)

	servicesClients.EXPECT().
		DoJSONRequest(gomock.Any(), "POST", "/task/bulk", gomock.Any(), gomock.Any()).
		DoAndReturn(
			func(ctx context.Context, method, path string, in interface{}, out interface{}) (http.Header, int, error) {
				actualHooks, ok := in.(map[string]sdk.NodeHook)
				require.True(t, ok)
				require.Len(t, actualHooks, 2)
				var repositoryWebHookFound, schedulerFound bool

				for k, h := range actualHooks {
					if h.HookModelName == "RepositoryWebHook" {
						repositoryWebHookFound = true
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
					if h.HookModelName == "Scheduler" {
						schedulerFound = true
						t.Logf("-->%T %+v", h, h)
						assert.Equal(t, "UTC", h.Config["timezone"].Value)
						assert.Equal(t, "0/15 * * * *", h.Config["cron"].Value)
						assert.Equal(t, `{
  "git.author": "",
  "git.branch": "master",
  "git.hash": "",
  "git.hash.before": "",
  "git.message": "",
  "git.repository": "fsamin/go-repo",
  "git.tag": ""
}`, h.Config["payload"].Value)
					}
				}
				assert.True(t, repositoryWebHookFound)
				assert.True(t, schedulerFound)

				out = actualHooks
				return nil, 200, nil
			},
		).Times(1)

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
		).Times(1)

	servicesClients.EXPECT().
		DoJSONRequest(gomock.Any(), "POST", "/vcs/github/repos/fsamin/go-repo/grant", gomock.Any(), gomock.Any(), gomock.Any())

	servicesClients.EXPECT().
		DoJSONRequest(gomock.Any(), "POST", "/vcs/github/repos/fsamin/go-repo/hooks", gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(
			func(ctx context.Context, method, path string, in interface{}, out interface{}, _ interface{}) (http.Header, int, error) {
				log.Debug("--> %T %+v", in, in)

				vcsHooks, ok := in.(*sdk.VCSHook)
				require.True(t, ok)

				require.Len(t, vcsHooks.Events, 0, "events list should be empty, default value is set by vcs")
				vcsHooks.Events = []string{"push"}

				assert.Equal(t, "http://lolcat.host", vcsHooks.URL)
				vcsHooks.ID = sdk.UUID()
				*(out.(*sdk.VCSHook)) = *vcsHooks

				return nil, 200, nil
			},
		).Times(1)

	// Perform a "import as-code operation" to create a new workflow
	ope := sdk.Operation{
		RepoFullName: "fsamin/go-repo",
		VCSServer:    "github",
		URL:          "https://github.com/fsamin/go-repo.git",
		RepositoryStrategy: sdk.RepositoryStrategy{
			ConnectionType: "https",
			DefaultBranch:  "master",
		},
		Setup: sdk.OperationSetup{
			Checkout: sdk.OperationCheckout{
				Branch: "master",
			},
		},
	}
	uri := api.Router.GetRoute("POST", api.postImportAsCodeHandler, map[string]string{
		"permProjectKey": proj.Key,
	})
	req := assets.NewJWTAuthentifiedRequest(t, jwt, "POST", uri, ope)
	w := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w, req)
	require.Equal(t, 201, w.Code)
	// Fake wait api event for operation done
	time.Sleep(time.Second)

	uri = api.Router.GetRoute("POST", api.postPerformImportAsCodeHandler, map[string]string{
		"permProjectKey": prjKey,
		"uuid":           operationUUID,
	})
	req = assets.NewJWTAuthentifiedRequest(t, jwt, "POST", uri, nil)
	w = httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w, req)
	require.Equal(t, 200, w.Code)
	t.Logf(w.Body.String())

	wk, err := workflow.Load(context.Background(), db, api.Cache, *proj, "w-go-repo", workflow.LoadOptions{})
	require.NoError(t, err)
	require.NotNil(t, wk)
	require.Len(t, wk.WorkflowData.GetHooks(), 2)

	for _, h := range wk.WorkflowData.GetHooks() {
		log.Debug("--> %T %+v", h, h)

		switch h.HookModelName {
		case "RepositoryWebHook":
			assert.Equal(t, "push", h.Config["eventFilter"].Value)
			assert.Equal(t, "Github", h.Config["hookIcon"].Value)
			assert.Equal(t, "POST", h.Config["method"].Value)
			assert.Equal(t, proj.Key, h.Config["project"].Value)
			assert.Equal(t, "fsamin/go-repo", h.Config["repoFullName"].Value)
			assert.Equal(t, "github", h.Config["vcsServer"].Value)
			assert.Equal(t, wk.Name, h.Config["workflow"].Value)
			assert.NotEmpty(t, h.Config["webHookID"].Value)

		case "Scheduler":
			assert.Equal(t, "UTC", h.Config["timezone"].Value)
			assert.Equal(t, "0/15 * * * *", h.Config["cron"].Value)
			assert.Equal(t, `{
  "git.author": "",
  "git.branch": "master",
  "git.hash": "",
  "git.hash.before": "",
  "git.message": "",
  "git.repository": "fsamin/go-repo",
  "git.tag": ""
}`, h.Config["payload"].Value)

		default:
			t.Fatalf("unexpected hook: %T %+v", h, h)
		}
	}

	// Then we will trigger a run of the workflow wich should trigger an as-code operation
	uri = api.Router.GetRoute("POST", api.postWorkflowRunHandler, map[string]string{
		"key":              proj.Key,
		"permWorkflowName": wk.Name,
	})
	require.NotEmpty(t, uri)
	req = assets.NewAuthentifiedRequest(t, u, jwt, "POST", uri, &sdk.WorkflowRunPostHandlerOption{
		Hook: &sdk.WorkflowNodeRunHookEvent{
			WorkflowNodeHookUUID: wk.WorkflowData.Node.Hooks[0].UUID,
			Payload: map[string]string{
				"git.branch": "master",
			},
		},
	})
	rec := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	if !assert.Equal(t, 202, rec.Code) {
		t.Logf("body => %s", rec.Body.String())
		t.FailNow()
	}
	var wrun sdk.WorkflowRun
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &wrun))

	require.NoError(t, waitCraftinWorkflow(t, api, api.mustDB(), wrun.ID))

	wr, err := workflow.LoadRunByID(db, wrun.ID, workflow.LoadRunOptions{})
	require.NoError(t, err)
	require.NotEqual(t, "Fail", wr.Status)

	wk, err = workflow.Load(context.Background(), db, api.Cache, *proj, "w-go-repo", workflow.LoadOptions{})
	assert.NoError(t, err)
	require.Len(t, wk.WorkflowData.GetHooks(), 2)

	for _, h := range wk.WorkflowData.GetHooks() {
		switch h.HookModelName {
		case "RepositoryWebHook":
			assert.Equal(t, "push", h.Config["eventFilter"].Value)
			assert.Equal(t, "Github", h.Config["hookIcon"].Value)
			assert.Equal(t, "POST", h.Config["method"].Value)
			assert.Equal(t, proj.Key, h.Config["project"].Value)
			assert.Equal(t, "fsamin/go-repo", h.Config["repoFullName"].Value)
			assert.Equal(t, "github", h.Config["vcsServer"].Value)
			assert.Equal(t, wk.Name, h.Config["workflow"].Value)
			assert.NotEmpty(t, h.Config["webHookID"].Value)

		case "Scheduler":
			assert.Equal(t, "UTC", h.Config["timezone"].Value)
			assert.Equal(t, "0/15 * * * *", h.Config["cron"].Value)
			assert.Equal(t, `{
  "git.author": "",
  "git.branch": "master",
  "git.hash": "",
  "git.hash.before": "",
  "git.message": "",
  "git.repository": "fsamin/go-repo",
  "git.tag": ""
}`, h.Config["payload"].Value)

		default:
			t.Fatalf("unexpected hook: %T %+v", h, h)
		}
	}
}

func Test_WorkflowAsCodeWithJustAcheduler_ShouldGive_ARepoWebHookAndTheScheduler_ThenWeUpdateIt_ThenWheDeleteIt(t *testing.T) {
	// 1. We create a new workflow from a repository in which there is a .cds directory
	// We check that all the as-code operation is successfull and create the right hooks
	// 2. Then we trigger the workflow, this aims to run another "as-code operation"
	// This time, the scheduler should be updated
	// This should not change the repowebhook
	// 3. Next we trigger the workflow, this aims to run another "as-code operation"
	// This last time, the scheduler should be deleted
	// This should not change the repowebhook

	api, db, _ := newTestAPI(t)

	_, _ = assets.InsertService(t, db, t.Name()+"_HOOKS", sdk.TypeHooks)
	_, _ = assets.InsertService(t, db, t.Name()+"_VCS", sdk.TypeVCS)
	_, _ = assets.InsertService(t, db, t.Name()+"_REPO", sdk.TypeRepositories)

	// Setup a mock for all services called by the API
	ctrl := gomock.NewController(t)
	t.Cleanup(func() { ctrl.Finish() })
	// The mock has been geenrated by mockgen: go get github.com/golang/mock/mockgen
	// If you have to regenerate thi mock you just have to run, from directory $GOPATH/src/github.com/ovh/cds/engine/api/services:
	// mockgen -source=http.go -destination=mock_services/services_mock.go Client
	servicesClients := mock_services.NewMockClient(ctrl)
	services.NewClient = func(_ gorp.SqlExecutor, _ []sdk.Service) services.Client { return servicesClients }
	t.Cleanup(func() { services.NewClient = services.NewDefaultClient })

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
			}).Times(3)

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
		).Times(5)

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
		).MaxTimes(5)

	gomock.InOrder(
		servicesClients.EXPECT().
			DoJSONRequest(gomock.Any(), "GET", "/operations/"+UUID, gomock.Any(), gomock.Any()).
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
pipeline_hooks:
- type: Scheduler
  config:
    cron: 0/15 * * * *
    payload: |-
      {
        "git.author": "",
        "git.branch": "master",
        "git.hash": "",
        "git.hash.before": "",
        "git.message": "",
        "git.repository": "fsamin/go-repo",
        "git.tag": ""
      }
    timezone: UTC
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
name: w-go-repo
version: v1.0
pipeline: build
application: go-repo
pipeline_hooks:
- type: Scheduler
  config:
    cron: 0/5 * * * *
    payload: |-
      {
        "git.author": "",
        "git.branch": "master",
        "git.hash": "",
        "git.hash.before": "",
        "git.message": "",
        "git.repository": "fsamin/go-repo",
        "git.tag": ""
      }
    timezone: UTC
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
			).Times(1),
	)

	gomock.InOrder(
		servicesClients.EXPECT().
			DoJSONRequest(gomock.Any(), "POST", "/task/bulk", gomock.Any(), gomock.Any()).
			DoAndReturn(
				func(ctx context.Context, method, path string, in interface{}, out interface{}) (http.Header, int, error) {
					actualHooks, ok := in.(map[string]sdk.NodeHook)
					require.True(t, ok)
					require.Len(t, actualHooks, 2)
					var repositoryWebHookFound, schedulerFound bool

					for k, h := range actualHooks {
						if h.HookModelName == "RepositoryWebHook" {
							repositoryWebHookFound = true
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
						if h.HookModelName == "Scheduler" {
							schedulerFound = true
							t.Logf("-->%T %+v", h, h)
							assert.Equal(t, "UTC", h.Config["timezone"].Value)
							assert.Equal(t, "0/15 * * * *", h.Config["cron"].Value)
							assert.Equal(t, `{
  "git.author": "",
  "git.branch": "master",
  "git.hash": "",
  "git.hash.before": "",
  "git.message": "",
  "git.repository": "fsamin/go-repo",
  "git.tag": ""
}`, h.Config["payload"].Value)
						}
					}
					assert.True(t, repositoryWebHookFound)
					assert.True(t, schedulerFound)

					out = actualHooks
					return nil, 200, nil
				},
			).Times(1),

		servicesClients.EXPECT().
			DoJSONRequest(gomock.Any(), "POST", "/task/bulk", gomock.Any(), gomock.Any()).
			DoAndReturn(
				func(ctx context.Context, method, path string, in interface{}, out interface{}) (http.Header, int, error) {
					actualHooks, ok := in.(map[string]sdk.NodeHook)
					require.True(t, ok)
					require.Len(t, actualHooks, 1)
					for _, h := range actualHooks {
						assert.Equal(t, "Scheduler", h.HookModelName)
						t.Logf("-->%T %+v", h, h)
						assert.Equal(t, "UTC", h.Config["timezone"].Value)
						assert.Equal(t, "0/5 * * * *", h.Config["cron"].Value)
						assert.Equal(t, `{
  "git.author": "",
  "git.branch": "master",
  "git.hash": "",
  "git.hash.before": "",
  "git.message": "",
  "git.repository": "fsamin/go-repo",
  "git.tag": ""
}`, h.Config["payload"].Value)
					}
					return nil, 200, nil
				},
			).Times(1),

		servicesClients.EXPECT().
			DoJSONRequest(gomock.Any(), "DELETE", "/task/bulk", gomock.Any(), gomock.Any()).
			DoAndReturn(
				func(ctx context.Context, method, path string, in interface{}, out interface{}) (http.Header, int, error) {
					actualHooks, ok := in.(map[string]sdk.NodeHook)
					require.True(t, ok)
					require.Len(t, actualHooks, 1)
					for _, h := range actualHooks {
						assert.Equal(t, "Scheduler", h.HookModelName)
						t.Logf("-->%T %+v", h, h)
						assert.Equal(t, "UTC", h.Config["timezone"].Value)
						assert.Equal(t, "0/15 * * * *", h.Config["cron"].Value)
						assert.Equal(t, `{
  "git.author": "",
  "git.branch": "master",
  "git.hash": "",
  "git.hash.before": "",
  "git.message": "",
  "git.repository": "fsamin/go-repo",
  "git.tag": ""
}`, h.Config["payload"].Value)
					}
					return nil, 200, nil
				},
			).Times(1),

		servicesClients.EXPECT().
			DoJSONRequest(gomock.Any(), "DELETE", "/task/bulk", gomock.Any(), gomock.Any()).
			DoAndReturn(
				func(ctx context.Context, method, path string, in interface{}, out interface{}) (http.Header, int, error) {
					actualHooks, ok := in.(map[string]sdk.NodeHook)
					require.True(t, ok)
					require.Len(t, actualHooks, 1)
					for _, h := range actualHooks {
						assert.Equal(t, "Scheduler", h.HookModelName)
						t.Logf("-->%T %+v", h, h)
						assert.Equal(t, "UTC", h.Config["timezone"].Value)
						assert.Equal(t, "0/5 * * * *", h.Config["cron"].Value)
						assert.Equal(t, `{
  "git.author": "",
  "git.branch": "master",
  "git.hash": "",
  "git.hash.before": "",
  "git.message": "",
  "git.repository": "fsamin/go-repo",
  "git.tag": ""
}`, h.Config["payload"].Value)
					}
					return nil, 200, nil
				},
			).Times(1),
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
		).Times(1)

	servicesClients.EXPECT().
		DoJSONRequest(gomock.Any(), "POST", "/vcs/github/repos/fsamin/go-repo/grant", gomock.Any(), gomock.Any(), gomock.Any())

	servicesClients.EXPECT().
		DoJSONRequest(gomock.Any(), "POST", "/vcs/github/repos/fsamin/go-repo/hooks", gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(
			func(ctx context.Context, method, path string, in interface{}, out interface{}, _ interface{}) (http.Header, int, error) {
				log.Debug("--> %T %+v", in, in)

				vcsHooks, ok := in.(*sdk.VCSHook)
				require.True(t, ok)

				require.Len(t, vcsHooks.Events, 0, "events list should be empty, default value is set by vcs")
				vcsHooks.Events = []string{"push"}

				assert.Equal(t, "POST", vcsHooks.Method)
				assert.Equal(t, "http://lolcat.host", vcsHooks.URL)
				vcsHooks.ID = sdk.UUID()
				*(out.(*sdk.VCSHook)) = *vcsHooks
				return nil, 200, nil
			},
		).Times(1)

	// ================================================================
	// 1. Perform a "import as-code operation" to create a new workflow
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

	// Do the request
	w = httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w, req)
	if !assert.Equal(t, 200, w.Code) {
		t.Logf(w.Body.String())
		t.FailNow()
	}

	wk, err := workflow.Load(context.Background(), db, api.Cache, *proj, "w-go-repo", workflow.LoadOptions{})
	require.NoError(t, err)
	require.Len(t, wk.WorkflowData.GetHooks(), 2)

	var repositoryWebHookFound, schedulerFound bool
	for _, h := range wk.WorkflowData.GetHooks() {
		log.Debug("--> %T %+v", h, h)

		switch h.HookModelName {
		case "RepositoryWebHook":
			assert.Equal(t, "push", h.Config["eventFilter"].Value)
			assert.Equal(t, "Github", h.Config["hookIcon"].Value)
			assert.Equal(t, "POST", h.Config["method"].Value)
			assert.Equal(t, proj.Key, h.Config["project"].Value)
			assert.Equal(t, "fsamin/go-repo", h.Config["repoFullName"].Value)
			assert.Equal(t, "github", h.Config["vcsServer"].Value)
			assert.Equal(t, wk.Name, h.Config["workflow"].Value)
			assert.NotEmpty(t, h.Config["webHookID"].Value)
			repositoryWebHookFound = true
		case "Scheduler":
			assert.Equal(t, "UTC", h.Config["timezone"].Value)
			assert.Equal(t, "0/15 * * * *", h.Config["cron"].Value)
			assert.Equal(t, `{
  "git.author": "",
  "git.branch": "master",
  "git.hash": "",
  "git.hash.before": "",
  "git.message": "",
  "git.repository": "fsamin/go-repo",
  "git.tag": ""
}`, h.Config["payload"].Value)
			schedulerFound = true
		default:
			t.Fatalf("unexpected hook: %T %+v", h, h)
		}
	}

	require.True(t, repositoryWebHookFound)
	require.True(t, schedulerFound)

	if t.Failed() {
		t.FailNow()
	}

	// ======================================================================================
	// 2. Then we will trigger a run of the workflow wich should trigger an as-code operation
	uri = api.Router.GetRoute("POST", api.postWorkflowRunHandler, map[string]string{
		"key":              proj.Key,
		"permWorkflowName": wk.Name,
	})
	require.NotEmpty(t, uri)
	req = assets.NewAuthentifiedRequest(t, u, pass, "POST", uri, &sdk.WorkflowRunPostHandlerOption{
		Hook: &sdk.WorkflowNodeRunHookEvent{
			WorkflowNodeHookUUID: wk.WorkflowData.Node.Hooks[1].UUID,
			Payload: map[string]string{
				"git.branch": "master",
			},
		},
	})
	rec := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	if !assert.Equal(t, 202, rec.Code) {
		t.Logf("body => %s", rec.Body.String())
		t.FailNow()
	}
	var wrun sdk.WorkflowRun
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &wrun))

	require.NoError(t, waitCraftinWorkflow(t, api, api.mustDB(), wrun.ID))

	wr, err := workflow.LoadRunByID(db, wrun.ID, workflow.LoadRunOptions{})
	require.NoError(t, err)

	assert.NotEqual(t, "Fail", wr.Status)

	wk, err = workflow.Load(context.Background(), db, api.Cache, *proj, "w-go-repo", workflow.LoadOptions{})
	require.NoError(t, err)
	require.Len(t, wk.WorkflowData.GetHooks(), 2)

	for _, h := range wk.WorkflowData.GetHooks() {
		switch h.HookModelName {
		case "RepositoryWebHook":
			assert.Equal(t, "push", h.Config["eventFilter"].Value)
			assert.Equal(t, "Github", h.Config["hookIcon"].Value)
			assert.Equal(t, "POST", h.Config["method"].Value)
			assert.Equal(t, proj.Key, h.Config["project"].Value)
			assert.Equal(t, "fsamin/go-repo", h.Config["repoFullName"].Value)
			assert.Equal(t, "github", h.Config["vcsServer"].Value)
			assert.Equal(t, wk.Name, h.Config["workflow"].Value)

		case "Scheduler":
			assert.Equal(t, "UTC", h.Config["timezone"].Value)
			assert.Equal(t, "0/5 * * * *", h.Config["cron"].Value)
			assert.Equal(t, `{
  "git.author": "",
  "git.branch": "master",
  "git.hash": "",
  "git.hash.before": "",
  "git.message": "",
  "git.repository": "fsamin/go-repo",
  "git.tag": ""
}`, h.Config["payload"].Value)

		default:
			t.Fatalf("unexpected hook: %T %+v", h, h)
		}
	}

	// ===========================================================================================================
	// 3. Then we will trigger a run of the workflow wich should trigger an as-code operation with a hook deletion
	uri = api.Router.GetRoute("POST", api.postWorkflowRunHandler, map[string]string{
		"key":              proj.Key,
		"permWorkflowName": wk.Name,
	})
	require.NotEmpty(t, uri)

	req = assets.NewAuthentifiedRequest(t, u, pass, "POST", uri, &sdk.WorkflowRunPostHandlerOption{
		Hook: &sdk.WorkflowNodeRunHookEvent{
			WorkflowNodeHookUUID: wk.WorkflowData.Node.Hooks[1].UUID,
			Payload: map[string]string{
				"git.branch": "master",
			},
		},
	})

	//Do the request
	rec = httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	if !assert.Equal(t, 202, rec.Code) {
		t.Logf("body => %s", rec.Body.String())
		t.FailNow()
	}

	var wrun2 sdk.WorkflowRun
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &wrun2))

	require.NoError(t, waitCraftinWorkflow(t, api, api.mustDB(), wrun2.ID))

	wr, err = workflow.LoadRunByID(db, wrun2.ID, workflow.LoadRunOptions{})
	require.NoError(t, err)
	require.NotEqual(t, "Fail", wr.Status)

	wk, err = workflow.Load(context.Background(), db, api.Cache, *proj, "w-go-repo", workflow.LoadOptions{})
	require.NoError(t, err)
	require.Len(t, wk.WorkflowData.GetHooks(), 1)

	for _, h := range wk.WorkflowData.GetHooks() {
		switch h.HookModelName {
		case "RepositoryWebHook":
			assert.Equal(t, "push", h.Config["eventFilter"].Value)
			assert.Equal(t, "Github", h.Config["hookIcon"].Value)
			assert.Equal(t, "POST", h.Config["method"].Value)
			assert.Equal(t, proj.Key, h.Config["project"].Value)
			assert.Equal(t, "fsamin/go-repo", h.Config["repoFullName"].Value)
			assert.Equal(t, "github", h.Config["vcsServer"].Value)
			assert.Equal(t, wk.Name, h.Config["workflow"].Value)

		default:
			t.Fatalf("unexpected hook: %T %+v", h, h)
		}
	}
}
