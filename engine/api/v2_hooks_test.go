package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/ovh/cds/engine/api/entity"
	"github.com/ovh/cds/engine/api/services"
	"github.com/ovh/cds/engine/api/services/mock_services"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/engine/api/user"
	"github.com/ovh/cds/engine/api/workflow_v2"
	"github.com/ovh/cds/sdk"
	"github.com/stretchr/testify/require"
)

func TestPostRetrieveEventUserHandler(t *testing.T) {
	api, db, _ := newTestAPI(t)
	api.Config.VCS.GPGKeys = make(map[string][]GPGKey)

	admin, pwd := assets.InsertAdminUser(t, db)
	require.NoError(t, user.InsertGPGKey(context.TODO(), db, &sdk.UserGPGKey{KeyID: "AZERTY", AuthentifiedUserID: admin.ID}))
	p := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))
	vcs := assets.InsertTestVCSProject(t, db, p.ID, "github", sdk.VCSTypeGithub)

	signKeyRequest := sdk.HookRetrieveUserRequest{
		ProjectKey:     p.Key,
		VCSServerName:  vcs.Name,
		RepositoryName: "myrepo",
		Commit:         "123",
		VCSServerType:  "github",
		SignKey:        "AZERTY",
	}

	uri := api.Router.GetRouteV2("POST", api.postRetrieveEventUserHandler, nil)
	test.NotEmpty(t, uri)
	req := assets.NewAuthentifiedRequest(t, nil, pwd, "POST", uri, &signKeyRequest)
	w := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w, req)
	require.Equal(t, 200, w.Code)

	var signKeyResponse sdk.HookRetrieveUserResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &signKeyResponse))
	require.Equal(t, admin.ID, signKeyResponse.UserID)

}
func TestPostHookEventRetrieveSignKeyHandler(t *testing.T) {
	api, db, _ := newTestAPI(t)

	_, pwd := assets.InsertAdminUser(t, db)

	p := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))
	vcs := assets.InsertTestVCSProject(t, db, p.ID, "github", sdk.VCSTypeGithub)

	// Mock VCS
	sVCS, _ := assets.InsertService(t, db, t.Name()+"_VCS", sdk.TypeVCS)
	sRepo, _ := assets.InsertService(t, db, t.Name()+"_REPO", sdk.TypeRepositories)
	sHooks, _ := assets.InsertService(t, db, t.Name()+"_HOOKS", sdk.TypeHooks)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	servicesClients := mock_services.NewMockClient(ctrl)
	services.NewClient = func(_ []sdk.Service) services.Client {
		return servicesClients
	}
	defer func() {
		_ = services.Delete(db, sVCS)
		_ = services.Delete(db, sRepo)
		_ = services.Delete(db, sHooks)
		services.NewClient = services.NewDefaultClient
	}()

	servicesClients.EXPECT().
		DoJSONRequest(gomock.Any(), "GET", "/vcs/github/repos/myrepo", gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(
			func(ctx context.Context, method, path string, in interface{}, out interface{}, _ interface{}) (http.Header, int, error) {
				repo := sdk.VCSRepo{}
				*(out.(*sdk.VCSRepo)) = repo
				return nil, 200, nil
			},
		).MaxTimes(1)
	servicesClients.EXPECT().
		DoJSONRequest(gomock.Any(), "POST", "/operations", gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, method, path string, in interface{}, out interface{}, mods ...interface{}) (http.Header, int, error) {
			ope := new(sdk.Operation)
			ope.UUID = "111-111-111"
			ope.Status = sdk.OperationStatusPending
			*(out.(*sdk.Operation)) = *ope
			return nil, 201, nil
		}).Times(1)

	servicesClients.EXPECT().DoJSONRequest(gomock.Any(), "GET", "/operations/111-111-111", gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, method, path string, in interface{}, out interface{}, mods ...interface{}) (http.Header, int, error) {
			ope := new(sdk.Operation)
			ope.UUID = "111-111-111"
			ope.Status = sdk.OperationStatusDone
			*(out.(*sdk.Operation)) = *ope
			return nil, 201, nil
		}).Times(1)

	servicesClients.EXPECT().DoJSONRequest(gomock.Any(), "POST", "/v2/repository/event/callback", gomock.Any(), gomock.Any()).Times(1)

	signKeyRequest := sdk.HookRetrieveSignKeyRequest{
		ProjectKey:     p.Key,
		VCSServerName:  vcs.Name,
		RepositoryName: "myrepo",
		Commit:         "123",
		Ref:            "refs/heads/master",
		HookEventUUID:  "123456",
		VCSServerType:  "github",
	}
	uri := api.Router.GetRouteV2("POST", api.postHookEventRetrieveSignKeyHandler, nil)
	test.NotEmpty(t, uri)
	req := assets.NewAuthentifiedRequest(t, nil, pwd, "POST", uri, &signKeyRequest)
	w := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w, req)
	require.Equal(t, 200, w.Code)

	time.Sleep(2 * time.Second)
}

func TestPostRetrieveWorkflowToTriggerHandler_RepositoryWebHooks(t *testing.T) {
	api, db, _ := newTestAPI(t)

	_, err := db.Exec("DELETE FROM v2_workflow_hook")
	require.NoError(t, err)

	_, pwd := assets.InsertAdminUser(t, db)

	p := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))
	vcs := assets.InsertTestVCSProject(t, db, p.ID, "github", sdk.VCSTypeGithub)
	repo := assets.InsertTestProjectRepository(t, db, p.Key, vcs.ID, sdk.RandomString(10))
	e := sdk.Entity{
		Name:                "MyWorkflow",
		Type:                sdk.EntityTypeWorkflow,
		ProjectKey:          p.Key,
		ProjectRepositoryID: repo.ID,
		Commit:              "123456",
		Ref:                 "refs/heads/master",
	}
	require.NoError(t, entity.Insert(context.TODO(), db, &e))

	wh1 := sdk.V2WorkflowHook{
		ProjectKey:     p.Key,
		VCSName:        vcs.Name,
		RepositoryName: repo.Name,
		EntityID:       e.ID,
		WorkflowName:   sdk.RandomString(10),
		Commit:         "123456",
		Ref:            "refs/heads/master",
		Type:           sdk.WorkflowHookTypeRepository,
		Data: sdk.V2WorkflowHookData{
			RepositoryName:  repo.Name,
			VCSServer:       vcs.Name,
			RepositoryEvent: sdk.WorkflowHookEventPush,
		},
	}
	require.NoError(t, workflow_v2.InsertWorkflowHook(context.TODO(), db, &wh1))

	r := sdk.HookListWorkflowRequest{
		RepositoryName:      repo.Name,
		VCSName:             vcs.Name,
		RepositoryEventName: sdk.WorkflowHookEventPush,
		AnayzedProjectKeys:  []string{p.Key},
		Ref:                 "refs/heads/master",
		Sha:                 "123456",
	}

	uri := api.Router.GetRouteV2("POST", api.postRetrieveWorkflowToTriggerHandler, nil)
	test.NotEmpty(t, uri)
	req := assets.NewAuthentifiedRequest(t, nil, pwd, "POST", uri, &r)
	w := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w, req)
	require.Equal(t, 200, w.Code)

	var hs []sdk.V2WorkflowHook
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &hs))

	require.Equal(t, 1, len(hs))
	require.Equal(t, wh1.ID, hs[0].ID)
}

func TestPostRetrieveWorkflowToTriggerHandler_RepositoryWebHooksPullRequest(t *testing.T) {
	api, db, _ := newTestAPI(t)

	_, err := db.Exec("DELETE FROM v2_workflow_hook")
	require.NoError(t, err)

	_, pwd := assets.InsertAdminUser(t, db)

	p := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))
	vcs := assets.InsertTestVCSProject(t, db, p.ID, "github", sdk.VCSTypeGithub)
	repo := assets.InsertTestProjectRepository(t, db, p.Key, vcs.ID, sdk.RandomString(10))
	e := sdk.Entity{
		Name:                "MyWorkflow",
		Type:                sdk.EntityTypeWorkflow,
		ProjectKey:          p.Key,
		ProjectRepositoryID: repo.ID,
		Commit:              "123456",
		Ref:                 "refs/heads/master",
	}
	require.NoError(t, entity.Insert(context.TODO(), db, &e))

	wh1 := sdk.V2WorkflowHook{
		ProjectKey:     p.Key,
		VCSName:        vcs.Name,
		RepositoryName: repo.Name,
		EntityID:       e.ID,
		WorkflowName:   sdk.RandomString(10),
		Commit:         "123456",
		Ref:            "refs/heads/master",
		Type:           sdk.WorkflowHookTypeRepository,
		Data: sdk.V2WorkflowHookData{
			RepositoryName:  repo.Name,
			VCSServer:       vcs.Name,
			RepositoryEvent: sdk.WorkflowHookEventPullRequest,
			TypesFilter:     []string{"pr:opened"},
		},
	}
	require.NoError(t, workflow_v2.InsertWorkflowHook(context.TODO(), db, &wh1))

	r := sdk.HookListWorkflowRequest{
		RepositoryName:      repo.Name,
		VCSName:             vcs.Name,
		RepositoryEventName: sdk.WorkflowHookEventPullRequest,
		RepositoryEventType: "pr:opened",
		Ref:                 "refs/heads/master",
		Sha:                 "123456",
		AnayzedProjectKeys:  []string{p.Key},
	}

	uri := api.Router.GetRouteV2("POST", api.postRetrieveWorkflowToTriggerHandler, nil)
	test.NotEmpty(t, uri)
	req := assets.NewAuthentifiedRequest(t, nil, pwd, "POST", uri, &r)
	w := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w, req)
	require.Equal(t, 200, w.Code)

	var hs []sdk.V2WorkflowHook
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &hs))

	require.Equal(t, 1, len(hs))
	require.Equal(t, wh1.ID, hs[0].ID)
}

func TestPostRetrieveWorkflowToTriggerHandler_RepositoryWebHooksPullRequestFiltered(t *testing.T) {
	api, db, _ := newTestAPI(t)

	_, err := db.Exec("DELETE FROM v2_workflow_hook")
	require.NoError(t, err)

	_, pwd := assets.InsertAdminUser(t, db)

	p := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))
	vcs := assets.InsertTestVCSProject(t, db, p.ID, "github", sdk.VCSTypeGithub)
	repo := assets.InsertTestProjectRepository(t, db, p.Key, vcs.ID, sdk.RandomString(10))
	e := sdk.Entity{
		Name:                "MyWorkflow",
		Type:                sdk.EntityTypeWorkflow,
		ProjectKey:          p.Key,
		ProjectRepositoryID: repo.ID,
		Commit:              "123456",
		Ref:                 "refs/heads/master",
	}
	require.NoError(t, entity.Insert(context.TODO(), db, &e))

	wh1 := sdk.V2WorkflowHook{
		ProjectKey:   p.Key,
		VCSName:      vcs.Name,
		EntityID:     e.ID,
		WorkflowName: sdk.RandomString(10),
		Commit:       "123456",
		Ref:          "refs/heads/master",
		Type:         sdk.WorkflowHookTypeRepository,
		Data: sdk.V2WorkflowHookData{
			RepositoryName:  repo.Name,
			VCSServer:       vcs.Name,
			RepositoryEvent: sdk.WorkflowHookEventPullRequest,
			TypesFilter:     []string{"pr:opened"},
		},
	}
	require.NoError(t, workflow_v2.InsertWorkflowHook(context.TODO(), db, &wh1))

	r := sdk.HookListWorkflowRequest{
		RepositoryName:      repo.Name,
		VCSName:             vcs.Name,
		RepositoryEventName: sdk.WorkflowHookEventPullRequest,
		RepositoryEventType: "pr:foo",
		AnayzedProjectKeys:  []string{p.Key},
	}

	uri := api.Router.GetRouteV2("POST", api.postRetrieveWorkflowToTriggerHandler, nil)
	test.NotEmpty(t, uri)
	req := assets.NewAuthentifiedRequest(t, nil, pwd, "POST", uri, &r)
	w := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w, req)
	require.Equal(t, 200, w.Code)

	var hs []sdk.V2WorkflowHook
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &hs))

	require.Equal(t, 0, len(hs))
}

func TestPostRetrieveWorkflowToTriggerHandler_WorkerModels(t *testing.T) {
	api, db, _ := newTestAPI(t)

	_, err := db.Exec("DELETE FROM v2_workflow_hook")
	require.NoError(t, err)

	_, pwd := assets.InsertAdminUser(t, db)

	p := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))
	vcs := assets.InsertTestVCSProject(t, db, p.ID, "github", sdk.VCSTypeGithub)
	repo := assets.InsertTestProjectRepository(t, db, p.Key, vcs.ID, sdk.RandomString(10))
	e := sdk.Entity{
		Name:                "MyWorkflow",
		Type:                sdk.EntityTypeWorkflow,
		ProjectKey:          p.Key,
		ProjectRepositoryID: repo.ID,
		Commit:              "123456",
		Ref:                 "refs/heads/master",
	}
	require.NoError(t, entity.Insert(context.TODO(), db, &e))

	wh1 := sdk.V2WorkflowHook{
		ProjectKey:   p.Key,
		VCSName:      vcs.Name,
		EntityID:     e.ID,
		WorkflowName: sdk.RandomString(10),
		Commit:       "123456",
		Ref:          "refs/heads/master",
		Type:         sdk.WorkflowHookTypeWorkerModel,
		Data: sdk.V2WorkflowHookData{
			Model: fmt.Sprintf("%s/%s/%s/%s", p.Key, vcs.Name, repo.Name, "MyModel"),
		},
	}
	require.NoError(t, workflow_v2.InsertWorkflowHook(context.TODO(), db, &wh1))

	r := sdk.HookListWorkflowRequest{
		RepositoryName:      repo.Name,
		VCSName:             vcs.Name,
		Ref:                 "refs/heads/master",
		Sha:                 "123456",
		RepositoryEventName: sdk.WorkflowHookEventPush,
		AnayzedProjectKeys:  []string{p.Key},
		Models: []sdk.EntityFullName{
			{
				Name:       "MySuperModel",
				VCSName:    vcs.Name,
				RepoName:   repo.Name,
				ProjectKey: p.Key,
				Ref:        "refs/heads/master",
			},
			{
				Name:       "MyUnusedModel",
				VCSName:    vcs.Name,
				RepoName:   repo.Name,
				ProjectKey: p.Key,
				Ref:        "refs/heads/master",
			},
			{
				Name:       "MyModel",
				VCSName:    vcs.Name,
				RepoName:   repo.Name,
				ProjectKey: p.Key,
				Ref:        "refs/heads/master",
			},
		},
	}

	uri := api.Router.GetRouteV2("POST", api.postRetrieveWorkflowToTriggerHandler, nil)
	test.NotEmpty(t, uri)
	req := assets.NewAuthentifiedRequest(t, nil, pwd, "POST", uri, &r)
	w := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w, req)
	require.Equal(t, 200, w.Code)

	var hs []sdk.V2WorkflowHook
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &hs))

	require.Equal(t, 1, len(hs))
	require.Equal(t, wh1.ID, hs[0].ID)
}
