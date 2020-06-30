package api

import (
	"context"
	"encoding/json"
	"github.com/go-gorp/gorp"
	"github.com/golang/mock/gomock"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/repositoriesmanager"
	"github.com/ovh/cds/engine/api/services"
	"github.com/ovh/cds/engine/api/services/mock_services"
	"github.com/ovh/cds/engine/api/workflow"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/authentication"
	"github.com/ovh/cds/engine/api/authentication/builtin"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient"
)

func Test_postApplicationMetadataHandler_AsProvider(t *testing.T) {
	api, tsURL := newTestServer(t)

	u, _ := assets.InsertAdminUser(t, api.mustDB())
	localConsumer, err := authentication.LoadConsumerByTypeAndUserID(context.TODO(), api.mustDB(), sdk.ConsumerLocal, u.ID, authentication.LoadConsumerOptions.WithAuthentifiedUser)
	require.NoError(t, err)
	_, jws, err := builtin.NewConsumer(context.TODO(), api.mustDB(), sdk.RandomString(10), sdk.RandomString(10), localConsumer, u.GetGroupIDs(),
		sdk.NewAuthConsumerScopeDetails(sdk.AuthConsumerScopeProject))

	pkey := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, api.mustDB(), api.Cache, pkey, pkey)
	require.NoError(t, group.InsertLinkGroupUser(context.TODO(), api.mustDB(), &group.LinkGroupUser{
		GroupID:            proj.ProjectGroups[0].Group.ID,
		AuthentifiedUserID: u.ID,
		Admin:              true,
	}))

	//Insert Application
	app := &sdk.Application{
		Name: sdk.RandomString(10),
		Metadata: map[string]string{
			"a1": "a1",
		},
	}
	if err := application.Insert(api.mustDB(), *proj, app); err != nil {
		t.Fatal(err)
	}

	sdkclient := cdsclient.NewProviderClient(cdsclient.ProviderConfig{
		Host:  tsURL,
		Token: jws,
	})

	test.NoError(t, sdkclient.ApplicationMetadataUpdate(pkey, app.Name, "b1", "b1"))
	app, err = application.LoadByName(api.mustDB(), pkey, app.Name)
	test.NoError(t, err)
	assert.Equal(t, "a1", app.Metadata["a1"])
	assert.Equal(t, "b1", app.Metadata["b1"])

	apps, err := sdkclient.ApplicationsList(pkey, cdsclient.FilterByUser(u.Username), cdsclient.FilterByWritablePermission())
	test.NoError(t, err)
	assert.Equal(t, 1, len(apps))
}

func TestUpdateAsCodeApplicationHandler(t *testing.T) {
	api, db, _ := newTestAPI(t)

	u, pass := assets.InsertAdminUser(t, db)

	UUID := sdk.UUID()

	svcs, errS := services.LoadAll(context.TODO(), db)
	assert.NoError(t, errS)
	for _, s := range svcs {
		_ = services.Delete(db, &s) // nolint
	}

	_, _ = assets.InsertService(t, db, t.Name()+"_HOOKS", sdk.TypeHooks)
	_, _ = assets.InsertService(t, db, t.Name()+"_VCS", sdk.TypeVCS)
	_, _ = assets.InsertService(t, db, t.Name()+"_REPO", sdk.TypeRepositories)

	// Setup a mock for all services called by the API
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	servicesClients := mock_services.NewMockClient(ctrl)
	services.NewClient = func(_ gorp.SqlExecutor, _ []sdk.Service) services.Client {
		return servicesClients
	}
	defer func() {
		services.NewClient = services.NewDefaultClient
	}()

	servicesClients.EXPECT().
		DoJSONRequest(gomock.Any(), "POST", "/task/bulk", gomock.Any(), gomock.Any()).
		Return(nil, 201, nil)

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
	servicesClients.EXPECT().DoJSONRequest(gomock.Any(), "GET", "/vcs/github/repos/foo/myrepo/branches", gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, method, path string, in interface{}, out interface{}, _ interface{}) (http.Header, int, error) {
			bs := []sdk.VCSBranch{}
			b := sdk.VCSBranch{
				DisplayID:    "master",
				LatestCommit: "aaaaaaa",
			}
			bs = append(bs, b)
			out = bs
			return nil, 200, nil
		}).Times(1)

	servicesClients.EXPECT().
		DoJSONRequest(gomock.Any(), "POST", "/vcs/github/repos/foo/myrepo/hooks", gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(
			func(ctx context.Context, method, path string, in interface{}, out interface{}, _ interface{}) (http.Header, int, error) {
				vcsHooks, _ := in.(*sdk.VCSHook)
				vcsHooks.Events = []string{"push"}
				vcsHooks.ID = sdk.UUID()
				*(out.(*sdk.VCSHook)) = *vcsHooks
				return nil, 200, nil
			},
		).Times(1)

	servicesClients.EXPECT().
		DoJSONRequest(gomock.Any(), "GET", "/vcs/github/repos/foo/myrepo", gomock.Any(), gomock.Any(), gomock.Any()).MinTimes(0)

	servicesClients.EXPECT().
		DoMultiPartRequest(gomock.Any(), "POST", "/operations", gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, method, path string, _ interface{}, in interface{}, out interface{}) (int, error) {
			ope := new(sdk.Operation)
			ope.UUID = UUID
			*(out.(*sdk.Operation)) = *ope
			return 200, nil
		}).Times(1)

	servicesClients.EXPECT().
		DoJSONRequest(gomock.Any(), "GET", "/vcs/github/repos/foo/myrepo/pullrequests?state=open", gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, method, path string, in interface{}, out interface{}, _ interface{}) (http.Header, int, error) {
			vcsPRs := []sdk.VCSPullRequest{}
			*(out.(*[]sdk.VCSPullRequest)) = vcsPRs
			return nil, 200, nil
		}).Times(1)

	servicesClients.EXPECT().
		DoJSONRequest(gomock.Any(), "POST", "/vcs/github/repos/foo/myrepo/pullrequests", gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, method, path string, in interface{}, out interface{}, _ interface{}) (http.Header, int, error) {
			vcsPR := sdk.VCSPullRequest{}
			vcsPR.URL = "myURL"
			*(out.(*sdk.VCSPullRequest)) = vcsPR
			return nil, 200, nil
		}).Times(1)

	servicesClients.EXPECT().
		DoJSONRequest(gomock.Any(), "GET", "/operations/"+UUID, gomock.Any(), gomock.Any(), gomock.Any()).
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

				*(out.(*sdk.Operation)) = *ope
				return nil, 200, nil
			},
		).Times(1)

	assert.NoError(t, workflow.CreateBuiltinWorkflowHookModels(db))

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

	pip := sdk.Pipeline{
		Name:           sdk.RandomString(10),
		ProjectID:      proj.ID,
		FromRepository: "myrepofrom",
	}
	assert.NoError(t, pipeline.InsertPipeline(db, &pip))

	pip.Stages = []sdk.Stage{
		{
			Name:       "mystage",
			BuildOrder: 1,
			Enabled:    true,
		},
	}

	app := sdk.Application{
		Name:               sdk.RandomString(10),
		ProjectID:          proj.ID,
		RepositoryFullname: "foo/myrepo",
		VCSServer:          "github",
		FromRepository:     "myrepofrom",
	}
	assert.NoError(t, application.Insert(db, *proj, &app))
	assert.NoError(t, repositoriesmanager.InsertForApplication(db, &app))

	repoModel, err := workflow.LoadHookModelByName(db, sdk.RepositoryWebHookModelName)
	assert.NoError(t, err)

	wk := initWorkflow(t, db, proj, &app, &pip, repoModel)
	wk.FromRepository = "myrepofrom"
	require.NoError(t, workflow.Insert(context.Background(), db, api.Cache, *proj, wk))

	uri := api.Router.GetRoute("PUT", api.updateAsCodeApplicationHandler, map[string]string{
		"permProjectKey":  proj.Key,
		"applicationName": app.Name,
	})
	req := assets.NewJWTAuthentifiedRequest(t, pass, "PUT", uri, app)
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

	cpt := 0
	for {
		if cpt >= 10 {
			t.Fail()
			return
		}

		// Get operation
		uriGET := api.Router.GetRoute("GET", api.getWorkflowAsCodeHandler, map[string]string{
			"key":              proj.Key,
			"permWorkflowName": wk.Name,
			"uuid":             myOpe.UUID,
		})
		reqGET, err := http.NewRequest("GET", uriGET, nil)
		test.NoError(t, err)
		assets.AuthentifyRequest(t, reqGET, u, pass)
		wrGet := httptest.NewRecorder()
		api.Router.Mux.ServeHTTP(wrGet, reqGET)
		assert.Equal(t, 200, wrGet.Code)
		myOpeGet := new(sdk.Operation)
		err = json.Unmarshal(wrGet.Body.Bytes(), myOpeGet)
		assert.NoError(t, err)

		if myOpeGet.Status < sdk.OperationStatusDone {
			cpt++
			time.Sleep(1 * time.Second)
			continue
		}
		test.NoError(t, json.Unmarshal(wrGet.Body.Bytes(), myOpeGet))
		assert.Equal(t, "myURL", myOpeGet.Setup.Push.PRLink)
		break
	}
}
