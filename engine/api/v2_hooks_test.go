package api

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/ovh/cds/engine/api/entity"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/engine/api/workflow_v2"
	"github.com/ovh/cds/sdk"
	"github.com/stretchr/testify/require"
	"net/http/httptest"
	"testing"
)

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
		Branch:              "master",
	}
	require.NoError(t, entity.Insert(context.TODO(), db, &e))

	wh1 := sdk.V2WorkflowHook{
		ProjectKey:   p.Key,
		VCSName:      vcs.Name,
		EntityID:     e.ID,
		WorkflowName: sdk.RandomString(10),
		Commit:       "123456",
		Branch:       "master",
		Type:         sdk.WorkflowHookTypeRepository,
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
		Branch:              "master",
	}
	require.NoError(t, entity.Insert(context.TODO(), db, &e))

	wh1 := sdk.V2WorkflowHook{
		ProjectKey:   p.Key,
		VCSName:      vcs.Name,
		EntityID:     e.ID,
		WorkflowName: sdk.RandomString(10),
		Commit:       "123456",
		Branch:       "master",
		Type:         sdk.WorkflowHookTypeWorkerModel,
		Data: sdk.V2WorkflowHookData{
			Model: fmt.Sprintf("%s/%s/%s/%s", p.Key, vcs.Name, repo.Name, "MyModel"),
		},
	}
	require.NoError(t, workflow_v2.InsertWorkflowHook(context.TODO(), db, &wh1))

	r := sdk.HookListWorkflowRequest{
		RepositoryName:      repo.Name,
		VCSName:             vcs.Name,
		Branch:              "master",
		RepositoryEventName: sdk.WorkflowHookEventPush,
		AnayzedProjectKeys:  []string{p.Key},
		Models: []sdk.EntityFullName{
			{
				Name:       "MySuperModel",
				VCSName:    vcs.Name,
				RepoName:   repo.Name,
				ProjectKey: p.Key,
				Branch:     "master",
			},
			{
				Name:       "MyUnusedModel",
				VCSName:    vcs.Name,
				RepoName:   repo.Name,
				ProjectKey: p.Key,
				Branch:     "master",
			},
			{
				Name:       "MyModel",
				VCSName:    vcs.Name,
				RepoName:   repo.Name,
				ProjectKey: p.Key,
				Branch:     "master",
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
