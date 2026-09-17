package api

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/ovh/cds/engine/api/entity"
	"github.com/ovh/cds/engine/api/services"
	"github.com/ovh/cds/engine/api/services/mock_services"
	"github.com/ovh/cds/engine/api/workflow_v2"
	"go.uber.org/mock/gomock"

	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient"
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
	services.NewClient = func(_ []sdk.Service) services.Client {
		return servicesClients
	}
	defer func() {
		_ = services.Delete(db, s)
		_ = services.Delete(db, sHooks)
		services.NewClient = services.NewDefaultClient
	}()

	servicesClients.EXPECT().DoJSONRequest(gomock.Any(), "DELETE", "/v2/repository/event/vcs-github/ovh%2Fcds", gomock.Any(), gomock.Any(), gomock.Any())
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

func TestLoadDistantHooksByProjectKey(t *testing.T) {
	api, db, _ := newTestAPI(t)

	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))
	vcsProj := assets.InsertTestVCSProject(t, db, proj.ID, "vcs-github", "github")
	repo := assets.InsertTestProjectRepository(t, db, proj.Key, vcsProj.ID, "ovh/workflows")

	otherProj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))
	otherVCS := assets.InsertTestVCSProject(t, db, otherProj.ID, "vcs-github", "github")
	otherRepo := assets.InsertTestProjectRepository(t, db, otherProj.Key, otherVCS.ID, "ovh/other")

	e := sdk.Entity{
		ID:                  sdk.UUID(),
		ProjectKey:          proj.Key,
		ProjectRepositoryID: repo.ID,
		Type:                sdk.EntityTypeWorkflow,
		Name:                "my-workflow",
		Ref:                 "refs/heads/master",
		Commit:              "123456",
		Head:                true,
	}
	require.NoError(t, entity.Insert(context.TODO(), db, &e))

	insertHook := func(projectKey, vcsName, repoName, hookType string, head bool, data sdk.V2WorkflowHookData) sdk.V2WorkflowHook {
		h := sdk.V2WorkflowHook{
			ProjectKey:     projectKey,
			VCSName:        vcsName,
			RepositoryName: repoName,
			EntityID:       e.ID,
			WorkflowName:   "my-workflow",
			Ref:            "refs/heads/master",
			Commit:         "123456",
			Type:           hookType,
			Head:           head,
			Data:           data,
		}
		require.NoError(t, workflow_v2.InsertWorkflowHook(context.TODO(), db, &h))
		return h
	}

	// Workflow and code on the same repository: not a distant hook
	insertHook(proj.Key, vcsProj.Name, repo.Name, sdk.WorkflowHookTypeRepository, true, sdk.V2WorkflowHookData{
		RepositoryEvent: sdk.WorkflowHookEventNamePush,
		VCSServer:       vcsProj.Name,
		RepositoryName:  repo.Name,
	})

	// Distant push hook: the target is lowercased by the analysis
	insertHook(proj.Key, vcsProj.Name, repo.Name, sdk.WorkflowHookTypeRepository, true, sdk.V2WorkflowHookData{
		RepositoryEvent: sdk.WorkflowHookEventNamePush,
		VCSServer:       vcsProj.Name,
		RepositoryName:  "ovh/distant",
	})

	// Scheduler hook on the same target, kept with its original case
	insertHook(proj.Key, vcsProj.Name, repo.Name, sdk.WorkflowHookTypeScheduler, true, sdk.V2WorkflowHookData{
		VCSServer:      vcsProj.Name,
		RepositoryName: "OVH/Distant",
	})

	// Not head anymore: the workflow does not listen to that repository
	insertHook(proj.Key, vcsProj.Name, repo.Name, sdk.WorkflowHookTypeRepository, false, sdk.V2WorkflowHookData{
		RepositoryEvent: sdk.WorkflowHookEventNamePush,
		VCSServer:       vcsProj.Name,
		RepositoryName:  "ovh/outdated",
	})

	// Distant hook of another project
	insertHook(otherProj.Key, otherVCS.Name, otherRepo.Name, sdk.WorkflowHookTypeRepository, true, sdk.V2WorkflowHookData{
		RepositoryEvent: sdk.WorkflowHookEventNamePush,
		VCSServer:       otherVCS.Name,
		RepositoryName:  "ovh/another-project-distant",
	})

	// Editing the repository the hook is attached to breaks its signature: the row must be ignored
	// instead of turning into a distant repository nobody declared.
	tampered := insertHook(proj.Key, vcsProj.Name, repo.Name, sdk.WorkflowHookTypeRepository, true, sdk.V2WorkflowHookData{
		RepositoryEvent: sdk.WorkflowHookEventNamePush,
		VCSServer:       vcsProj.Name,
		RepositoryName:  repo.Name,
	})
	_, err := db.Exec("UPDATE v2_workflow_hook SET repository_name = $1 WHERE id = $2", "ovh/tampered", tampered.ID)
	require.NoError(t, err)

	hooks, err := workflow_v2.LoadDistantHooksByProjectKey(context.TODO(), db, proj.Key)
	require.NoError(t, err)

	targets := make([]string, 0, len(hooks))
	for _, h := range hooks {
		targets = append(targets, h.Data.RepositoryName)
	}
	require.ElementsMatch(t, []string{"ovh/distant", "OVH/Distant"}, targets)
}

func Test_getProjectDistantRepositoryAllHandler(t *testing.T) {
	api, db, _ := newTestAPI(t)

	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))
	user1, pass := assets.InsertLambdaUser(t, db)
	assets.InsertRBAcProject(t, db, "read", proj.Key, *user1)

	vcsProj := assets.InsertTestVCSProject(t, db, proj.ID, "vcs-github", "github")
	repo := assets.InsertTestProjectRepository(t, db, proj.Key, vcsProj.ID, "ovh/workflows")
	assets.InsertTestProjectRepository(t, db, proj.Key, vcsProj.ID, "ovh/declared")

	e := sdk.Entity{
		ID:                  sdk.UUID(),
		ProjectKey:          proj.Key,
		ProjectRepositoryID: repo.ID,
		Type:                sdk.EntityTypeWorkflow,
		Name:                "my-workflow",
		Ref:                 "refs/heads/master",
		Commit:              "123456",
		Head:                true,
	}
	require.NoError(t, entity.Insert(context.TODO(), db, &e))

	insertDistantHook := func(targetRepo string) {
		h := sdk.V2WorkflowHook{
			ProjectKey:     proj.Key,
			VCSName:        vcsProj.Name,
			RepositoryName: repo.Name,
			EntityID:       e.ID,
			WorkflowName:   "my-workflow",
			Ref:            "refs/heads/master",
			Commit:         "123456",
			Type:           sdk.WorkflowHookTypeRepository,
			Head:           true,
			Data: sdk.V2WorkflowHookData{
				RepositoryEvent: sdk.WorkflowHookEventNamePush,
				VCSServer:       vcsProj.Name,
				RepositoryName:  targetRepo,
			},
		}
		require.NoError(t, workflow_v2.InsertWorkflowHook(context.TODO(), db, &h))
	}
	insertDistantHook("ovh/distant")
	// Same target, written with another case by another hook type: one entry expected
	insertDistantHook("OVH/Distant")
	// Already declared in the project, whatever the case used by the workflow definition
	insertDistantHook("OVH/Declared")
	// Second distant repository, to check the results are sorted
	insertDistantHook("ovh/another")

	callHandler := func() []sdk.ProjectDistantRepository {
		uri := api.Router.GetRouteV2("GET", api.getProjectDistantRepositoryAllHandler, map[string]string{"projectKey": proj.Key})
		test.NotEmpty(t, uri)
		req := assets.NewAuthentifiedRequest(t, user1, pass, "GET", uri, nil)
		w := httptest.NewRecorder()
		api.Router.Mux.ServeHTTP(w, req)
		require.Equal(t, 200, w.Code)
		var repositories []sdk.ProjectDistantRepository
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &repositories))
		return repositories
	}

	// The two hooks on ovh/distant belong to the same workflow: listed once
	listener := sdk.ProjectDistantRepositoryWorkflow{VCSName: vcsProj.Name, RepositoryName: repo.Name, WorkflowName: "my-workflow"}
	require.Equal(t, []sdk.ProjectDistantRepository{
		{VCSName: vcsProj.Name, Repository: "ovh/another", Workflows: []sdk.ProjectDistantRepositoryWorkflow{listener}},
		{VCSName: vcsProj.Name, Repository: "ovh/distant", Workflows: []sdk.ProjectDistantRepositoryWorkflow{listener}},
	}, callHandler())
}

func Test_getProjectRepositoryEventsHandler_DistantRepository(t *testing.T) {
	api, db, _ := newTestAPI(t)

	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))
	user1, pass := assets.InsertLambdaUser(t, db)
	assets.InsertRBAcProject(t, db, "read", proj.Key, *user1)
	vcsProj := assets.InsertTestVCSProject(t, db, proj.ID, "vcs-github", "github")
	assets.InsertTestProjectRepository(t, db, proj.Key, vcsProj.ID, "ovh/declared")

	sVCS, _ := assets.InsertService(t, db, t.Name()+"_VCS", sdk.TypeVCS)
	sHooks, _ := assets.InsertService(t, db, t.Name()+"_HOOKS", sdk.TypeHooks)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	servicesClients := mock_services.NewMockClient(ctrl)
	services.NewClient = func(_ []sdk.Service) services.Client {
		return servicesClients
	}
	defer func() {
		_ = services.Delete(db, sVCS)
		_ = services.Delete(db, sHooks)
		services.NewClient = services.NewDefaultClient
	}()

	listEvents := func(repoName string) *httptest.ResponseRecorder {
		uri := api.Router.GetRouteV2("GET", api.getProjectRepositoryEventsHandler, map[string]string{
			"projectKey":           proj.Key,
			"vcsIdentifier":        vcsProj.Name,
			"repositoryIdentifier": url.PathEscape(repoName),
		})
		test.NotEmpty(t, uri)
		req := assets.NewAuthentifiedRequest(t, user1, pass, "GET", uri, nil)
		w := httptest.NewRecorder()
		api.Router.Mux.ServeHTTP(w, req)
		return w
	}

	// Not declared but readable by the vcs server credentials: the events are served, keyed by the
	// lowercased repository name
	servicesClients.EXPECT().
		DoJSONRequest(gomock.Any(), "GET", "/vcs/vcs-github/repos/OVH/Distant", gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, method, path string, in interface{}, out interface{}, _ ...cdsclient.RequestModifier) (http.Header, int, error) {
			*(out.(*sdk.VCSRepo)) = sdk.VCSRepo{Fullname: "OVH/Distant"}
			return nil, 200, nil
		})
	servicesClients.EXPECT().
		DoJSONRequest(gomock.Any(), "GET", "/v2/repository/event/vcs-github/ovh%2Fdistant", gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, method, path string, in interface{}, out interface{}, _ ...cdsclient.RequestModifier) (http.Header, int, error) {
			*(out.(*[]sdk.HookRepositoryEvent)) = []sdk.HookRepositoryEvent{{UUID: "distant-event"}}
			return nil, 200, nil
		})
	w := listEvents("OVH/Distant")
	require.Equal(t, 200, w.Code)
	var events []sdk.HookRepositoryEvent
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &events))
	require.Len(t, events, 1)
	require.Equal(t, "distant-event", events[0].UUID)

	// Not declared and not readable: the vcs server answer is returned, hooks are never asked
	servicesClients.EXPECT().
		DoJSONRequest(gomock.Any(), "GET", "/vcs/vcs-github/repos/ovh/unreadable", gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil, 404, sdk.WithStack(sdk.ErrNotFound))
	require.Equal(t, 404, listEvents("ovh/unreadable").Code)

	// Declared: unchanged, the vcs server is never asked
	servicesClients.EXPECT().
		DoJSONRequest(gomock.Any(), "GET", "/v2/repository/event/vcs-github/ovh%2Fdeclared", gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, method, path string, in interface{}, out interface{}, _ ...cdsclient.RequestModifier) (http.Header, int, error) {
			*(out.(*[]sdk.HookRepositoryEvent)) = []sdk.HookRepositoryEvent{{UUID: "declared-event"}}
			return nil, 200, nil
		})
	w = listEvents("ovh/declared")
	require.Equal(t, 200, w.Code)
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &events))
	require.Len(t, events, 1)
	require.Equal(t, "declared-event", events[0].UUID)
}
