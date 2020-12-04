package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-gorp/gorp"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/event"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/repositoriesmanager"
	"github.com/ovh/cds/engine/api/services"
	"github.com/ovh/cds/engine/api/services/mock_services"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient"
	"github.com/ovh/cds/sdk/exportentities"
	"github.com/ovh/cds/sdk/log"
)

func TestPostUpdateWorkflowAsCodeHandler(t *testing.T) {
	api, db, tsURL := newTestServer(t)

	event.OverridePubSubKey("events_pubsub_test")
	require.NoError(t, event.Initialize(context.Background(), api.mustDB(), api.Cache))
	require.NoError(t, api.initWebsocket("events_pubsub_test"))

	u, jwt := assets.InsertAdminUser(t, db)

	client := cdsclient.New(cdsclient.Config{
		Host:                  tsURL,
		User:                  u.Username,
		InsecureSkipVerifyTLS: true,
		SessionToken:          jwt,
	})

	_, _ = assets.InsertService(t, db, t.Name()+"_HOOKS", sdk.TypeHooks)
	_, _ = assets.InsertService(t, db, t.Name()+"_VCS", sdk.TypeVCS)
	_, _ = assets.InsertService(t, db, t.Name()+"_REPO", sdk.TypeRepositories)

	UUID := sdk.UUID()

	svcs, errS := services.LoadAll(context.TODO(), db)
	assert.NoError(t, errS)
	for _, s := range svcs {
		_ = services.Delete(db, &s) // nolint
	}

	a, _ := assets.InsertService(t, db, "Test_postWorkflowAsCodeHandlerVCS", sdk.TypeVCS)
	b, _ := assets.InsertService(t, db, "Test_postWorkflowAsCodeHandlerRepo", sdk.TypeRepositories)
	c, _ := assets.InsertService(t, db, "Test_postWorkflowAsCodeHandlerHook", sdk.TypeHooks)

	defer func() {
		services.Delete(db, a)
		services.Delete(db, b)
		services.Delete(db, c)
	}()

	//This is a mock for the repositories service
	services.HTTPClient = mock(
		func(r *http.Request) (*http.Response, error) {
			body := new(bytes.Buffer)
			w := new(http.Response)
			enc := json.NewEncoder(body)
			w.Body = ioutil.NopCloser(body)
			w.StatusCode = http.StatusOK
			switch r.URL.String() {
			case "/operations":
				ope := new(sdk.Operation)
				ope.UUID = UUID
				ope.Status = sdk.OperationStatusProcessing
				if err := enc.Encode(ope); err != nil {
					return writeError(w, err)
				}
			case "/operations/" + UUID:
				ope := new(sdk.Operation)
				ope.UUID = UUID
				ope.Status = sdk.OperationStatusDone
				if err := enc.Encode(ope); err != nil {
					return writeError(w, err)
				}
			case "/vcs/github/repos/foo/myrepo":
				vcsRepo := sdk.VCSRepo{
					Name:         "foo/myrepo",
					SSHCloneURL:  "git:foo",
					HTTPCloneURL: "https:foo",
				}
				if err := enc.Encode(vcsRepo); err != nil {
					return writeError(w, err)
				}
			case "/vcs/github/repos/foo/myrepo/branches":
				bs := make([]sdk.VCSBranch, 1)
				bs[0] = sdk.VCSBranch{
					DisplayID: "master",
					Default:   true,
				}
				if err := enc.Encode(bs); err != nil {
					return writeError(w, err)
				}
			case "/vcs/github/webhooks":
				hookInfo := repositoriesmanager.WebhooksInfos{
					WebhooksSupported: true,
					WebhooksDisabled:  false,
				}
				if err := enc.Encode(hookInfo); err != nil {
					return writeError(w, err)
				}
			case "/vcs/github/repos/foo/myrepo/hooks":
				hook := sdk.VCSHook{
					ID: "myod",
				}
				if err := enc.Encode(hook); err != nil {
					return writeError(w, err)
				}
			case "/vcs/github/repos/foo/myrepo/pullrequests?state=open":
				vcsPRs := []sdk.VCSPullRequest{}
				if err := enc.Encode(vcsPRs); err != nil {
					return writeError(w, err)
				}
			case "/vcs/github/repos/foo/myrepo/pullrequests":
				vcsPR := sdk.VCSPullRequest{
					URL: "myURL",
				}
				if err := enc.Encode(vcsPR); err != nil {
					return writeError(w, err)
				}
			case "/task/bulk":
				var hooks map[string]sdk.NodeHook
				bts, err := ioutil.ReadAll(r.Body)
				if err != nil {
					return writeError(w, err)
				}
				if err := json.Unmarshal(bts, &hooks); err != nil {
					return writeError(w, err)
				}
				if err := enc.Encode(hooks); err != nil {
					return writeError(w, err)
				}
			default:
				t.Logf("[WRONG ROUTE] %s", r.URL.String())
				w.StatusCode = http.StatusNotFound
			}

			return w, nil
		},
	)

	require.NoError(t, workflow.CreateBuiltinWorkflowHookModels(api.mustDB()))

	proj := createProject(t, db, api)
	pip := createPipeline(t, db, api, proj)
	app := createApplication(t, db, api, proj)

	repoModel, err := workflow.LoadHookModelByName(db, sdk.RepositoryWebHookModelName)
	assert.NoError(t, err)

	w := initWorkflow(t, db, proj, app, pip, repoModel)
	w.FromRepository = "myfromrepositoryurl"

	var errP error
	proj, errP = project.Load(context.TODO(), api.mustDB(), proj.Key,
		project.LoadOptions.WithApplicationWithDeploymentStrategies,
		project.LoadOptions.WithPipelines,
		project.LoadOptions.WithEnvironments,
		project.LoadOptions.WithIntegrations,
	)
	assert.NoError(t, errP)
	if !assert.NoError(t, workflow.Insert(context.Background(), db, api.Cache, *proj, w)) {
		return
	}

	// Updating workflow
	w.WorkflowData.Node.Triggers = []sdk.NodeTrigger{
		{
			ChildNode: sdk.Node{
				Type: "fork",
				Name: "secondnode",
			},
		},
	}

	chanMessageReceived := make(chan sdk.WebsocketEvent)
	chanMessageToSend := make(chan []sdk.WebsocketFilter)
	chanErrorReceived := make(chan error)
	go client.WebsocketEventsListen(context.TODO(), sdk.NewGoRoutines(), chanMessageToSend, chanMessageReceived, chanErrorReceived)
	chanMessageToSend <- []sdk.WebsocketFilter{{
		Type:         sdk.WebsocketFilterTypeAscodeEvent,
		ProjectKey:   proj.Key,
		WorkflowName: w.Name,
	}}

	uri := api.Router.GetRoute("POST", api.postWorkflowAsCodeHandler, map[string]string{
		"key":              proj.Key,
		"permWorkflowName": w.Name,
	})

	req := assets.NewJWTAuthentifiedRequest(t, jwt, "POST", uri, w)
	q := req.URL.Query()
	q.Set("branch", "master")
	q.Set("message", "my message")
	req.URL.RawQuery = q.Encode()

	// Do the request
	wr := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(wr, req)
	assert.Equal(t, 200, wr.Code)
	myOpe := new(sdk.Operation)
	test.NoError(t, json.Unmarshal(wr.Body.Bytes(), myOpe))
	assert.NotEmpty(t, myOpe.UUID)

	timeout := time.NewTimer(5 * time.Second)
	select {
	case <-timeout.C:
		t.Fatal("test timeout")
	case evt := <-chanMessageReceived:
		require.Equal(t, fmt.Sprintf("%T", sdk.EventAsCodeEvent{}), evt.Event.EventType)
		var ae sdk.EventAsCodeEvent
		require.NoError(t, json.Unmarshal(evt.Event.Payload, &ae))
		require.Equal(t, "myURL", ae.Event.PullRequestURL)
		break
	}
}

func TestPostMigrateWorkflowAsCodeHandler(t *testing.T) {
	api, db, tsURL := newTestServer(t)

	event.OverridePubSubKey("events_pubsub_test")
	require.NoError(t, event.Initialize(context.Background(), api.mustDB(), api.Cache))
	require.NoError(t, api.initWebsocket("events_pubsub_test"))

	u, jwt := assets.InsertAdminUser(t, db)

	client := cdsclient.New(cdsclient.Config{
		Host:                  tsURL,
		User:                  u.Username,
		InsecureSkipVerifyTLS: true,
		SessionToken:          jwt,
	})

	UUID := sdk.UUID()

	a, _ := assets.InsertService(t, db, "Test_postWorkflowAsCodeHandlerVCS", sdk.TypeVCS)
	b, _ := assets.InsertService(t, db, "Test_postWorkflowAsCodeHandlerRepo", sdk.TypeRepositories)
	c, _ := assets.InsertService(t, db, "Test_postWorkflowAsCodeHandlerHook", sdk.TypeHooks)

	defer func() {
		_ = services.Delete(db, a)
		_ = services.Delete(db, b)
		_ = services.Delete(db, c)
	}()

	require.NoError(t, workflow.CreateBuiltinWorkflowHookModels(api.mustDB()))

	//This is a mock for the repositories service
	services.HTTPClient = mock(
		func(r *http.Request) (*http.Response, error) {
			body := new(bytes.Buffer)
			w := new(http.Response)
			enc := json.NewEncoder(body)
			w.Body = ioutil.NopCloser(body)
			w.StatusCode = http.StatusOK
			switch r.URL.String() {
			case "/operations":
				ope := new(sdk.Operation)
				ope.UUID = UUID
				ope.Status = sdk.OperationStatusProcessing
				if err := enc.Encode(ope); err != nil {
					return writeError(w, err)
				}
			case "/operations/" + UUID:
				ope := new(sdk.Operation)
				ope.UUID = UUID
				ope.Status = sdk.OperationStatusDone
				if err := enc.Encode(ope); err != nil {
					return writeError(w, err)
				}
			case "/vcs/github/repos/foo/myrepo":
				vcsRepo := sdk.VCSRepo{
					Name:         "foo/myrepo",
					SSHCloneURL:  "git:foo",
					HTTPCloneURL: "https:foo",
				}
				if err := enc.Encode(vcsRepo); err != nil {
					return writeError(w, err)
				}
			case "/vcs/github/repos/foo/myrepo/branches":
				bs := make([]sdk.VCSBranch, 1)
				bs[0] = sdk.VCSBranch{
					DisplayID: "master",
					Default:   true,
				}
				if err := enc.Encode(bs); err != nil {
					return writeError(w, err)
				}
			case "/vcs/github/webhooks":
				hookInfo := repositoriesmanager.WebhooksInfos{
					WebhooksSupported: true,
					WebhooksDisabled:  false,
				}
				if err := enc.Encode(hookInfo); err != nil {
					return writeError(w, err)
				}
			case "/vcs/github/repos/foo/myrepo/hooks":
				hook := sdk.VCSHook{
					ID: "myod",
				}
				if err := enc.Encode(hook); err != nil {
					return writeError(w, err)
				}
			case "/vcs/github/repos/foo/myrepo/pullrequests?state=open":
				vcsPRs := []sdk.VCSPullRequest{}
				if err := enc.Encode(vcsPRs); err != nil {
					return writeError(w, err)
				}
			case "/vcs/github/repos/foo/myrepo/pullrequests":
				vcsPR := sdk.VCSPullRequest{
					URL: "myURL",
				}
				if err := enc.Encode(vcsPR); err != nil {
					return writeError(w, err)
				}
			case "/task/bulk":
				var hooks map[string]sdk.NodeHook
				bts, err := ioutil.ReadAll(r.Body)
				if err != nil {
					return writeError(w, err)
				}
				if err := json.Unmarshal(bts, &hooks); err != nil {
					return writeError(w, err)
				}
				for k, h := range hooks {
					if h.HookModelName == sdk.RepositoryWebHookModelName {
						cfg := hooks[k].Config
						cfg["webHookURL"] = sdk.WorkflowNodeHookConfigValue{
							Value:        "http://lolcat.host",
							Configurable: false,
						}
					}
				}
				if err := enc.Encode(hooks); err != nil {
					return writeError(w, err)
				}
			default:
				t.Logf("[WRONG ROUTE] %s", r.URL.String())
				w.StatusCode = http.StatusNotFound
			}

			return w, nil
		},
	)

	// Create Project
	proj := createProject(t, db, api)

	// Create Pipeline
	pip := createPipeline(t, db, api, proj)

	// Create Application
	app := createApplication(t, db, api, proj)

	repoModel, err := workflow.LoadHookModelByName(db, sdk.RepositoryWebHookModelName)
	assert.NoError(t, err)

	// Create Workflow
	w := initWorkflow(t, db, proj, app, pip, repoModel)

	var errP error
	proj, errP = project.Load(context.TODO(), api.mustDB(), proj.Key,
		project.LoadOptions.WithApplicationWithDeploymentStrategies,
		project.LoadOptions.WithPipelines,
		project.LoadOptions.WithEnvironments,
		project.LoadOptions.WithIntegrations,
	)
	assert.NoError(t, errP)
	if !assert.NoError(t, workflow.Insert(context.Background(), db, api.Cache, *proj, w)) {
		return
	}

	t.Logf("%+v", w)

	chanMessageReceived := make(chan sdk.WebsocketEvent)
	chanMessageToSend := make(chan []sdk.WebsocketFilter)
	chanErrorReceived := make(chan error)
	go client.WebsocketEventsListen(context.TODO(), sdk.NewGoRoutines(), chanMessageToSend, chanMessageReceived, chanErrorReceived)
	chanMessageToSend <- []sdk.WebsocketFilter{{
		Type:         sdk.WebsocketFilterTypeAscodeEvent,
		ProjectKey:   proj.Key,
		WorkflowName: w.Name,
	}}

	uri := api.Router.GetRoute("POST", api.postWorkflowAsCodeHandler, map[string]string{
		"key":              proj.Key,
		"permWorkflowName": w.Name,
	})

	req := assets.NewJWTAuthentifiedRequest(t, jwt, "POST", uri, nil)
	q := req.URL.Query()
	q.Set("migrate", "true")
	q.Set("branch", "master")
	q.Set("message", "my message")
	req.URL.RawQuery = q.Encode()

	// Do the request
	wr := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(wr, req)
	assert.Equal(t, 200, wr.Code)
	myOpe := new(sdk.Operation)
	test.NoError(t, json.Unmarshal(wr.Body.Bytes(), myOpe))
	assert.NotEmpty(t, myOpe.UUID)

	timeout := time.NewTimer(5 * time.Second)
	select {
	case <-timeout.C:
		t.Fatal("test timeout")
	case evt := <-chanMessageReceived:
		require.Equal(t, fmt.Sprintf("%T", sdk.EventAsCodeEvent{}), evt.Event.EventType)
		var ae sdk.EventAsCodeEvent
		require.NoError(t, json.Unmarshal(evt.Event.Payload, &ae))
		require.Equal(t, "myURL", ae.Event.PullRequestURL)
		break
	}
}

func createProject(t *testing.T, db gorpmapper.SqlExecutorWithTx, api *API) *sdk.Project {
	// Create Project
	pkey := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, api.Cache, pkey, pkey)
	vcsServer := sdk.ProjectVCSServerLink{
		ProjectID: proj.ID,
		Name:      "github",
	}
	vcsServer.Set("token", "foo")
	vcsServer.Set("secret", "bar")
	assert.NoError(t, repositoriesmanager.InsertProjectVCSServerLink(context.TODO(), db, &vcsServer))

	return proj
}

func createPipeline(t *testing.T, db gorp.SqlExecutor, api *API, proj *sdk.Project) *sdk.Pipeline {
	pip := sdk.Pipeline{
		Name:      sdk.RandomString(10),
		ProjectID: proj.ID,
	}
	assert.NoError(t, pipeline.InsertPipeline(db, &pip))
	return &pip
}

func createApplication(t *testing.T, db gorpmapper.SqlExecutorWithTx, api *API, proj *sdk.Project) *sdk.Application {
	app := sdk.Application{
		Name:               sdk.RandomString(10),
		ProjectID:          proj.ID,
		RepositoryFullname: "foo/myrepo",
		VCSServer:          "github",
	}
	require.NoError(t, application.Insert(db, proj.ID, &app))
	require.NoError(t, repositoriesmanager.InsertForApplication(db, &app))
	return &app
}

func initWorkflow(t *testing.T, db gorp.SqlExecutor, proj *sdk.Project, app *sdk.Application, pip *sdk.Pipeline, repoModel *sdk.WorkflowHookModel) *sdk.Workflow {
	w := sdk.Workflow{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       sdk.RandomString(10),
		WorkflowData: sdk.WorkflowData{
			Node: sdk.Node{
				Name: "root",
				Type: sdk.NodeTypePipeline,
				Context: &sdk.NodeContext{
					PipelineID:    pip.ID,
					ApplicationID: app.ID,
				},
				Hooks: []sdk.NodeHook{
					{
						HookModelName: sdk.RepositoryWebHookModelName,
						UUID:          sdk.RandomString(10),
						Config:        sdk.RepositoryWebHookModel.DefaultConfig.Clone(),
						HookModelID:   repoModel.ID,
					},
				},
			},
		},
	}
	w.WorkflowData.Node.Hooks[0].Config["webHookURL"] = sdk.WorkflowNodeHookConfigValue{Type: "string", Value: "http://ovh/com"}
	assert.NoError(t, workflow.RenameNode(context.Background(), db, &w))
	return &w
}

func Test_WorkflowAsCodeWithNotifications(t *testing.T) {
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

	// Perform a "import as-code operation" to create a new workflow
	ope := `{"repo_fullname":"fsamin/go-repo",  "vcs_server": "github", "url":"https://github.com/fsamin/go-repo.git","strategy":{"connection_type":"https","ssh_key":"","user":"","password":"","branch":"","default_branch":"master","pgp_key":""},"setup":{"checkout":{"branch":"master"}}}`
	uri := api.Router.GetRoute("POST", api.postImportAsCodeHandler, map[string]string{
		"permProjectKey": proj.Key,
	})

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
					DisplayID:    "master",
					LatestCommit: "aaaaaaa",
				}
				bs = append(bs, b)
				out = bs
				return nil, 200, nil
			},
		).MaxTimes(10)

	servicesClients.EXPECT().
		DoJSONRequest(gomock.Any(), "GET", "/vcs/github/repos/fsamin/go-repo/branches/?branch=master", gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(
			func(ctx context.Context, method, path string, in interface{}, out interface{}, _ interface{}) (http.Header, int, error) {
				b := &sdk.VCSBranch{
					DisplayID:    "master",
					LatestCommit: "aaaaaaa",
				}
				*(out.(*sdk.VCSBranch)) = *b
				return nil, 200, nil
			},
		).MaxTimes(10)

	servicesClients.EXPECT().
		DoJSONRequest(gomock.Any(), "GET", "/vcs/github/repos/fsamin/go-repo/commits/aaaaaaa", gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(
			func(ctx context.Context, method, path string, in interface{}, out interface{}, _ interface{}) (http.Header, int, error) {
				c := &sdk.VCSCommit{
					Author: sdk.VCSAuthor{
						Avatar:      "me",
						DisplayName: "me",
						Name:        "me",
					},
					Hash:    "aaaaaaa",
					Message: "this is it",
				}
				*(out.(*sdk.VCSCommit)) = *c
				return nil, 200, nil
			},
		).MaxTimes(10)

	servicesClients.EXPECT().
		DoJSONRequest(gomock.Any(), "GET", "/vcs/github/repos/fsamin/go-repo/commits/aaaaaaa/statuses", gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(
			func(ctx context.Context, method, path string, in interface{}, out interface{}, _ interface{}) (http.Header, int, error) {
				return nil, 200, nil
			},
		).MaxTimes(10)

	servicesClients.EXPECT().
		DoJSONRequest(gomock.Any(), "GET", "/vcs/github", gomock.Any(), gomock.Any()).
		DoAndReturn(
			func(ctx context.Context, method, path string, in interface{}, out interface{}) (http.Header, int, error) {
				return nil, 200, nil
			},
		).MaxTimes(10)

	servicesClients.EXPECT().
		DoJSONRequest(gomock.Any(), "POST", "/vcs/github/status", gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(
			func(ctx context.Context, method, path string, in interface{}, out interface{}, _ interface{}) (http.Header, int, error) {
				return nil, 200, nil
			},
		).MaxTimes(10)

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
version: v2.0
workflow:
  playbook-game-molecule:
    depends_on:
    - root
    when:
    - success
    pipeline: build
    application: go-repo
  playbook-game-stuff:
    depends_on:
    - root
    when:
    - success
    pipeline: build
    application: go-repo
  netauto-versioning:
    depends_on:
    - release-prerequisites
    - playbook-game-molecule
    - playbook-game-stuff
    when:
    - success
    pipeline: build
    application: go-repo
  notify:
    depends_on:
    - netauto-versioning
    pipeline: build
    application: go-repo
  release-prerequisites:
    depends_on:
    - root
    when:
    - success
    pipeline: build
    application: go-repo
  root:
    pipeline: build
    application: go-repo
metadata:
  default_tags: tag_version,git.branch,git.author
notifications:
- type: vcs
- type: jabber
  pipelines:
  - playbook-game-stuff
  - playbook-game-molecule
  - netauto-versioning
  - release-prerequisites
  settings:
    on_success: never
    recipients:
    - ""
    template:
      subject: '{{.cds.project}}/{{.cds.workflow}}/{{.cds.pipeline}}#{{.cds.version}}
        {{.cds.status}}'
      body: |
        {{.cds.buildURL}}
- type: jabber
  pipelines:
  - notify
  settings:
    on_success: always
    on_failure: never
    recipients:
    - conf.netauto-release-mgmt@conference-2-standaloneclustere6001.corp.ovh.com
    template:
      subject: New release for {{.cds.workflow}}
      body: '{{.cds.workflow}} is now at version {{.workflow.netauto-versioning.build.VERSION}}'
    conditions:
      check:
      - variable: git.branch
        operator: eq
        value: master
- type: jabber
  pipelines:
  - notify
  settings:
    on_success: always
    on_failure: never

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

	req, err := http.NewRequest("POST", uri, strings.NewReader(ope))
	require.NoError(t, err)
	assets.AuthentifyRequest(t, req, u, pass)

	// Do the request
	w := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w, req)
	require.Equal(t, 201, w.Code)

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
	require.NotNil(t, wk)

	require.Len(t, wk.Notifications, 4, "not the right number of notifications")

	// Then we will trigger a run of the workflow wich should trigger an as-code operation
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

	wk, err = workflow.Load(context.Background(), db, api.Cache, *proj, "w-go-repo", workflow.LoadOptions{})
	assert.NoError(t, err)
	assert.NotNil(t, wk)

	require.Len(t, wk.Notifications, 4, "not the right number of notifications")
}

func Test_WorkflowAsCodeWithMultipleSchedulers(t *testing.T) {
	api, db, _ := newTestAPI(t)

	pip := `name: build
version: v1.0`

	app := `
name: go-repo
version: v1.0
repo: fsamin/go-repo
vcs_server: github
`
	wf := `
name: w-go-repo
version: v2.0
workflow:
  root:
    pipeline: build
    application: go-repo
metadata:
  default_tags: tag_version,git.branch,git.author
retention_policy: return run_days_before < 2
hooks:
  root:
  - type: RepositoryWebHook
  - type: Scheduler
    config:
      cron: 0 6,13 * * *
      payload: |-
        {
          "git.branch": "master",
          "scope": "all"
        }
      timezone: UTC
  - type: Scheduler
    config:
      cron: 0 7,13 * * *
      payload: |-
        {
          "git.branch": "master",
          "scope": "all2"
        }
      timezone: UTC
`

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
	u, _ := assets.InsertLambdaUser(t, db, &proj.ProjectGroups[0].Group)

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
			}).Times(1)

	servicesClients.EXPECT().
		DoJSONRequest(gomock.Any(), "GET", "/vcs/github/repos/fsamin/go-repo", gomock.Any(), gomock.Any(), gomock.Any()).MinTimes(0)

	servicesClients.EXPECT().
		DoJSONRequest(gomock.Any(), "GET", "/vcs/github/repos/fsamin/go-repo/branches", gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(
			func(ctx context.Context, method, path string, in interface{}, out interface{}, _ interface{}) (http.Header, int, error) {
				bs := []sdk.VCSBranch{}
				b := sdk.VCSBranch{
					DisplayID:    "master",
					LatestCommit: "aaaaaaa",
					Default:      true,
				}
				b2 := sdk.VCSBranch{
					DisplayID:    "non-default",
					LatestCommit: "aaaaaaa",
					Default:      false,
				}
				bs = append(bs, b, b2)
				out = bs
				return nil, 200, nil
			},
		).MaxTimes(10)

	servicesClients.EXPECT().
		DoJSONRequest(gomock.Any(), "GET", "/vcs/github/repos/fsamin/go-repo/branches/?branch=non-default", gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(
			func(ctx context.Context, method, path string, in interface{}, out interface{}, _ interface{}) (http.Header, int, error) {
				b := &sdk.VCSBranch{
					DisplayID:    "non-default",
					LatestCommit: "aaaaaaa",
					Default:      false,
				}
				*(out.(*sdk.VCSBranch)) = *b
				return nil, 200, nil
			},
		).MaxTimes(10)

	servicesClients.EXPECT().
		DoJSONRequest(gomock.Any(), "GET", "/vcs/github/repos/fsamin/go-repo/commits/aaaaaaa", gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(
			func(ctx context.Context, method, path string, in interface{}, out interface{}, _ interface{}) (http.Header, int, error) {
				c := &sdk.VCSCommit{
					Author: sdk.VCSAuthor{
						Avatar:      "me",
						DisplayName: "me",
						Name:        "me",
					},
					Hash:    "aaaaaaa",
					Message: "this is it",
				}
				*(out.(*sdk.VCSCommit)) = *c
				return nil, 200, nil
			},
		).MaxTimes(10)

	servicesClients.EXPECT().
		DoJSONRequest(gomock.Any(), "GET", "/vcs/github/repos/fsamin/go-repo/commits/aaaaaaa/statuses", gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(
			func(ctx context.Context, method, path string, in interface{}, out interface{}, _ interface{}) (http.Header, int, error) {
				return nil, 200, nil
			},
		).MaxTimes(10)

	servicesClients.EXPECT().
		DoJSONRequest(gomock.Any(), "GET", "/vcs/github", gomock.Any(), gomock.Any()).
		DoAndReturn(
			func(ctx context.Context, method, path string, in interface{}, out interface{}) (http.Header, int, error) {
				return nil, 200, nil
			},
		).MaxTimes(10)

	servicesClients.EXPECT().
		DoJSONRequest(gomock.Any(), "POST", "/vcs/github/status", gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(
			func(ctx context.Context, method, path string, in interface{}, out interface{}, _ interface{}) (http.Header, int, error) {
				return nil, 200, nil
			},
		).MaxTimes(10)

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
				ope.Setup.Checkout.Branch = "non-default"
				ope.RepositoryInfo = new(sdk.OperationRepositoryInfo)
				ope.RepositoryInfo.Name = "fsamin/go-repo"
				ope.RepositoryInfo.DefaultBranch = "master"
				ope.RepositoryInfo.FetchURL = "https://github.com/fsamin/go-repo.git"
				ope.LoadFiles.Pattern = workflow.WorkflowAsCodePattern
				ope.LoadFiles.Results = map[string][]byte{
					"w-go-repo.yml": []byte(`
name: w-go-repo
version: v2.0
workflow:
  root:
    pipeline: build
    application: go-repo
metadata:
  default_tags: tag_version,git.branch,git.author
retention_policy: return run_days_before < 2
hooks:
  root:
  - type: RepositoryWebHook
  - type: Scheduler
    config:
      cron: 0 6,12 * * *
      payload: |-
        {
          "git.branch": "master",
          "scope": "all"
        }
      timezone: UTC
  - type: Scheduler
    config:
      cron: 0 7,12 * * *
      payload: |-
        {
          "git.branch": "master",
          "scope": "all2"
        }
      timezone: UTC
`),
					"go-repo.app.yml": []byte(app),
					"go-repo.pip.yml": []byte(pip),
				}
				*(out.(*sdk.Operation)) = *ope
				return nil, 200, nil
			},
		).Times(1)

	servicesClients.EXPECT().
		DoJSONRequest(gomock.Any(), "POST", "/task/bulk", gomock.Any(), gomock.Any()).
		DoAndReturn(
			func(ctx context.Context, method, path string, in interface{}, out interface{}) (http.Header, int, error) {
				actualHooks, ok := in.(map[string]sdk.NodeHook)
				require.True(t, ok)

				for k, h := range actualHooks {
					if h.HookModelName == sdk.RepositoryWebHookModelName {
						h.Config["webHookURL"] = sdk.WorkflowNodeHookConfigValue{
							Value:        "http://lolcat.host",
							Configurable: false,
						}
						actualHooks[k] = h
					}
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

	// Import Pipeline
	pipData, err := exportentities.ParsePipeline(exportentities.FormatYAML, []byte(pip))
	require.NoError(t, err)
	pipDB, _, err := pipeline.ParseAndImport(context.TODO(), db, api.Cache, *proj, pipData, u, pipeline.ImportOptions{Force: true, PipelineName: "build", FromRepository: "https://github.com/fsamin/go-repo.git"})
	require.NoError(t, err)
	require.Equal(t, pipDB.FromRepository, "https://github.com/fsamin/go-repo.git")

	// Import Application
	var eapp exportentities.Application
	require.NoError(t, err, yaml.Unmarshal([]byte(app), &eapp))
	appDB, _, _, err := application.ParseAndImport(context.TODO(), db, api.Cache, *proj, &eapp, application.ImportOptions{Force: true, FromRepository: "https://github.com/fsamin/go-repo.git"}, nil, u)
	require.NoError(t, err)
	require.Equal(t, appDB.FromRepository, "https://github.com/fsamin/go-repo.git")

	// Import Workflow
	ew, err := exportentities.UnmarshalWorkflow([]byte(wf), exportentities.FormatYAML)
	require.NoError(t, err)
	workflowInserted, _, err := workflow.ParseAndImport(context.TODO(), db, api.Cache, *proj, nil, ew, u, workflow.ImportOptions{Force: true, FromRepository: "https://github.com/fsamin/go-repo.git"})
	require.NoError(t, err)
	require.Equal(t, workflowInserted.FromRepository, "https://github.com/fsamin/go-repo.git")

	opts := sdk.WorkflowRunPostHandlerOption{
		Hook: &sdk.WorkflowNodeRunHookEvent{
			WorkflowNodeHookUUID: workflowInserted.WorkflowData.Node.Hooks[0].UUID,
			Payload: map[string]string{
				"git.branch": "non-default",
			},
		},
	}
	_, _, err = workflow.CreateFromRepository(context.TODO(), api.mustDB(), api.Cache, proj, workflowInserted, opts, sdk.AuthConsumer{AuthentifiedUser: u}, project.DecryptWithBuiltinKey)
	require.NoError(t, err)
}
