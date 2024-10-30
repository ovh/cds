package api

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/ovh/cds/engine/api/entity"
	"github.com/ovh/cds/engine/api/repository"
	"github.com/ovh/cds/engine/api/services"
	"github.com/ovh/cds/engine/api/services/mock_services"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/engine/api/vcs"
	"github.com/ovh/cds/engine/api/workflow_v2"
	"github.com/ovh/cds/sdk"
	"github.com/stretchr/testify/require"
)

func TestCleanWorkflowVersionNoVCS(t *testing.T) {
	api, db, _ := newTestAPI(t)
	projKey := sdk.RandomString(10)
	assets.InsertTestProject(t, db, api.Cache, projKey, projKey)
	for i := 0; i < 50; i++ {
		v := sdk.V2WorkflowVersion{
			Version:            fmt.Sprintf("1.0.%d", i),
			ProjectKey:         projKey,
			WorkflowVCS:        "vcs",
			WorkflowRepository: "repo",
			WorkflowRef:        "ref",
			WorkflowSha:        "sha",
			VCSServer:          "server",
			Repository:         "repository",
			WorkflowName:       "name",
			WorkflowRunID:      sdk.UUID(),
			Username:           "username",
			UserID:             "111",
			Sha:                "sha",
			Ref:                "ref",
			Type:               "cargo",
			File:               "file",
		}
		require.NoError(t, workflow_v2.InsertWorkflowVersion(context.TODO(), db, &v))
	}
	err := workerCleanWorkflowVersion(context.TODO(), db.DbMap, api.Cache, workflow_v2.V2WorkflowVersionWorkflowShort{
		DistinctID:         "",
		ProjectKey:         projKey,
		WorkflowVCS:        "vcs",
		WorkflowRepository: "repo",
		WorkflowName:       "name",
	}, 14)
	require.NoError(t, err)

	versionsDB, err := workflow_v2.LoadAllVerionsByWorkflow(context.TODO(), db, projKey, "vcs", "repo", "name")
	require.NoError(t, err)
	require.Equal(t, 0, len(versionsDB))
}

func TestCleanWorkflowVersion(t *testing.T) {
	api, db, _ := newTestAPI(t)
	projKey := sdk.RandomString(10)

	proj := assets.InsertTestProject(t, db, api.Cache, projKey, projKey)
	vcsProj := assets.InsertTestVCSProject(t, db, proj.ID, "vcs", "github")
	repo := assets.InsertTestProjectRepository(t, db, projKey, vcsProj.ID, "repo")

	e := sdk.Entity{
		ID:                  sdk.UUID(),
		ProjectKey:          projKey,
		ProjectRepositoryID: repo.ID,
		Type:                sdk.EntityTypeWorkflow,
		Name:                "name",
		Ref:                 "refs/heads/master",
		Commit:              "HEAD",
	}
	require.NoError(t, entity.Insert(context.TODO(), db, &e))

	for i := 0; i < 50; i++ {
		v := sdk.V2WorkflowVersion{
			Version:            fmt.Sprintf("1.0.%d", i),
			ProjectKey:         projKey,
			WorkflowVCS:        "vcs",
			WorkflowRepository: "repo",
			WorkflowRef:        "ref",
			WorkflowSha:        "sha",
			VCSServer:          "server",
			Repository:         "repository",
			WorkflowName:       "name",
			WorkflowRunID:      sdk.UUID(),
			Username:           "username",
			UserID:             "111",
			Sha:                "sha",
			Ref:                "ref",
			Type:               "cargo",
			File:               "file",
		}
		require.NoError(t, workflow_v2.InsertWorkflowVersion(context.TODO(), db, &v))
	}

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
		DoJSONRequest(gomock.Any(), "GET", "/vcs/vcs/repos/repo/branches/?branch=&default=true", gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(
			func(ctx context.Context, method, path string, in interface{}, out interface{}, _ interface{}) (http.Header, int, error) {
				branch := sdk.VCSBranch{
					ID:        "refs/heads/master",
					DisplayID: "master",
				}
				*(out.(*sdk.VCSBranch)) = branch
				return nil, 200, nil
			},
		).MaxTimes(1)

	err := workerCleanWorkflowVersion(context.TODO(), db.DbMap, api.Cache, workflow_v2.V2WorkflowVersionWorkflowShort{
		DistinctID:         "",
		ProjectKey:         projKey,
		WorkflowVCS:        "vcs",
		WorkflowRepository: "repo",
		WorkflowName:       "name",
	}, 14)
	require.NoError(t, err)

	versionsDB, err := workflow_v2.LoadAllVerionsByWorkflow(context.TODO(), db, projKey, "vcs", "repo", "name")
	require.NoError(t, err)
	require.Equal(t, 14, len(versionsDB))
}

func Test_cleanAsCodeEntities(t *testing.T) {
	api, db, _ := newTestAPI(t)

	// Create project
	p := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))

	// Create VCS
	vcsProject := &sdk.VCSProject{
		Name:        "the-name",
		Type:        sdk.VCSTypeGithub,
		Auth:        sdk.VCSAuthProject{Username: "the-username", Token: "the-token"},
		Description: "the-username",
		ProjectID:   p.ID,
	}
	err := vcs.Insert(context.TODO(), db, vcsProject)
	require.NoError(t, err)
	require.NotEmpty(t, vcsProject.ID)

	// Create repository
	repo := sdk.ProjectRepository{
		Name:         "myrepo",
		Created:      time.Now(),
		VCSProjectID: vcsProject.ID,
		CreatedBy:    "me",
		CloneURL:     "myurl",
		ProjectKey:   p.Key,
	}
	require.NoError(t, repository.Insert(context.TODO(), db, &repo))

	wkfDelete := sdk.Entity{
		Name:                "model1",
		Commit:              "123456",
		Ref:                 "refs/heads/temp",
		Type:                sdk.EntityTypeWorkflow,
		ProjectRepositoryID: repo.ID,
		ProjectKey:          p.Key,
		Data:                `name: workflow1`,
	}
	require.NoError(t, entity.Insert(context.TODO(), db, &wkfDelete))

	etoDelete := sdk.Entity{
		Name:                "model1",
		Commit:              "123456",
		Ref:                 "refs/heads/temp",
		Type:                sdk.EntityTypeWorkerModel,
		ProjectRepositoryID: repo.ID,
		ProjectKey:          p.Key,
		Data: `name: model1
type: docker
osarch: linux/amd64
spec:
  image: monimage`,
	}
	require.NoError(t, entity.Insert(context.TODO(), db, &etoDelete))

	etoKeep := sdk.Entity{
		Name:                "model2",
		Commit:              "987654",
		Ref:                 "refs/heads/master",
		Type:                sdk.EntityTypeWorkerModel,
		ProjectRepositoryID: repo.ID,
		ProjectKey:          p.Key,
		Data: `name: model1
type: docker
osarch: linux/amd64
spec:
  image: monimage`,
	}
	require.NoError(t, entity.Insert(context.TODO(), db, &etoKeep))

	// Mock VCS
	s, _ := assets.InsertService(t, db, t.Name()+"_VCS", sdk.TypeVCS)
	sHook, _ := assets.InsertService(t, db, t.Name()+"_VCS", sdk.TypeHooks)
	// Setup a mock for all services called by the API
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	servicesClients := mock_services.NewMockClient(ctrl)
	services.NewClient = func(_ []sdk.Service) services.Client {
		return servicesClients
	}
	defer func() {
		_ = services.Delete(db, s)
		_ = services.Delete(db, sHook)
		services.NewClient = services.NewDefaultClient
	}()

	servicesClients.EXPECT().
		DoJSONRequest(gomock.Any(), "GET", "/vcs/the-name/repos/myrepo/branches?limit=100&noCache=true", gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(
			func(ctx context.Context, method, path string, in interface{}, out interface{}, _ interface{}) (http.Header, int, error) {
				branches := []sdk.VCSBranch{
					{
						ID:        "refs/heads/master",
						DisplayID: "master",
					},
				}
				*(out.(*[]sdk.VCSBranch)) = branches
				return nil, 200, nil
			},
		).MaxTimes(1)
	servicesClients.EXPECT().
		DoJSONRequest(gomock.Any(), "GET", "/vcs/the-name/repos/myrepo/tags", gomock.Any(), gomock.Any(), gomock.Any()).MaxTimes(1)

	err = workerCleanProject(context.TODO(), db.DbMap, api.Cache, p.Key, time.Minute)
	require.NoError(t, err)

	_, err = entity.LoadByRefTypeNameCommit(context.TODO(), db, repo.ID, "refs/heads/temp", sdk.EntityTypeWorkerModel, "model1", "123456")
	require.True(t, sdk.ErrorIs(err, sdk.ErrNotFound))

	e, err := entity.LoadByRefTypeNameCommit(context.TODO(), db, repo.ID, "refs/heads/master", sdk.EntityTypeWorkerModel, "model2", "987654")
	require.NoError(t, err)
	require.Equal(t, etoKeep.ID, e.ID)
}
