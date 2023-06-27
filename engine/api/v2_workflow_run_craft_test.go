package api

import (
	"context"
	"fmt"
	"github.com/go-gorp/gorp"
	"github.com/golang/mock/gomock"
	"github.com/ovh/cds/engine/api/entity"
	"github.com/ovh/cds/engine/api/services"
	"github.com/ovh/cds/engine/api/services/mock_services"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/engine/api/workflow_v2"
	"github.com/ovh/cds/sdk"
	"github.com/stretchr/testify/require"
	"net/http"
	"testing"
	"time"
)

func TestCraftWorkflowRunDepsNotFound(t *testing.T) {
	api, db, _ := newTestAPI(t)
	ctx := context.TODO()

	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))
	admin, _ := assets.InsertAdminUser(t, db)

	vcsProject := assets.InsertTestVCSProject(t, db, proj.ID, "github", "github")
	repo := assets.InsertTestProjectRepository(t, db, vcsProject.ID, "my/repo")

	wkName := sdk.RandomString(10)
	wr := sdk.V2WorkflowRun{
		UserID:       admin.ID,
		ProjectKey:   proj.Key,
		Status:       sdk.StatusCrafting,
		VCSServerID:  vcsProject.ID,
		RepositoryID: repo.ID,
		RunNumber:    0,
		RunAttempt:   0,
		WorkflowRef:  "master",
		WorkflowSha:  "123456",
		WorkflowName: wkName,
		WorkflowData: sdk.V2WorkflowRunData{
			Workflow: sdk.V2Workflow{
				Name: wkName,
				Jobs: map[string]sdk.V2Job{
					"job1": {
						Name:        "My super job",
						If:          "cds.workflow == 'toto'",
						Region:      "build",
						WorkerModel: "myworker-model",
						Steps: []sdk.ActionStep{
							{
								ID:   "myfirstStep",
								Uses: fmt.Sprintf("actions/%s/%s/%s/myaction", proj.Key, vcsProject.Name, repo.Name),
							},
							{
								ID:   "mysecondStep",
								Uses: fmt.Sprintf("actions/%s/%s/myaction", vcsProject.Name, repo.Name),
							},
							{
								ID:   "mythirdStep",
								Uses: fmt.Sprintf("actions/%s/myaction", repo.Name),
							},
							{
								ID:   "myfourthStep",
								Uses: fmt.Sprintf("actions/myaction"),
							},
						},
					},
				},
			},
		},
	}
	require.NoError(t, workflow_v2.InsertRun(ctx, db, &wr))

	require.NoError(t, api.craftWorkflowRunV2(ctx, wr.ID))

	wrDB, err := workflow_v2.LoadRunByID(ctx, db, wr.ID)
	require.NoError(t, err)
	require.Equal(t, sdk.StatusSkipped, wrDB.Status)
	wrInfos, err := workflow_v2.LoadRunInfosByRunID(ctx, db, wr.ID)
	require.NoError(t, err)
	require.Equal(t, 1, len(wrInfos))
	require.Equal(t, "obsolete workflow dependency used: myworker-model", wrInfos[0].Message)
}

func TestCraftWorkflowRunDepsSameRepo(t *testing.T) {
	api, db, _ := newTestAPI(t)
	ctx := context.TODO()

	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))
	admin, _ := assets.InsertAdminUser(t, db)

	vcsProject := assets.InsertTestVCSProject(t, db, proj.ID, "github", "github")
	repo := assets.InsertTestProjectRepository(t, db, vcsProject.ID, "my/repo")

	wkName := sdk.RandomString(10)
	wr := sdk.V2WorkflowRun{
		UserID:       admin.ID,
		ProjectKey:   proj.Key,
		Status:       sdk.StatusCrafting,
		VCSServerID:  vcsProject.ID,
		RepositoryID: repo.ID,
		RunNumber:    0,
		RunAttempt:   0,
		WorkflowRef:  "master",
		WorkflowSha:  "123456",
		WorkflowName: wkName,
		WorkflowData: sdk.V2WorkflowRunData{
			Workflow: sdk.V2Workflow{
				Name: wkName,
				Jobs: map[string]sdk.V2Job{
					"job1": {
						Name:        "My super job",
						If:          "cds.workflow == 'toto'",
						Region:      "build",
						WorkerModel: "myworker-model",
						Steps: []sdk.ActionStep{
							{
								ID:   "myfirstStep",
								Uses: fmt.Sprintf("actions/%s/%s/%s/myaction", proj.Key, vcsProject.Name, repo.Name),
							},
							{
								ID:   "mysecondStep",
								Uses: fmt.Sprintf("actions/%s/%s/myaction@master", vcsProject.Name, repo.Name),
							},
							{
								ID:   "mythirdStep",
								Uses: fmt.Sprintf("actions/%s/myaction", repo.Name),
							},
							{
								ID:   "myfourthStep",
								Uses: fmt.Sprintf("actions/myaction"),
							},
						},
					},
				},
			},
		},
	}
	require.NoError(t, workflow_v2.InsertRun(ctx, db, &wr))

	myactionEnt := sdk.Entity{
		ProjectKey:          proj.Key,
		ProjectRepositoryID: repo.ID,
		Type:                sdk.EntityTypeAction,
		FilePath:            ".cds/actions/myaction.yml",
		Name:                "myaction",
		Branch:              "master",
		Commit:              "123456",
		LastUpdate:          time.Time{},
		Data:                "name: myaction",
	}
	require.NoError(t, entity.Insert(ctx, db, &myactionEnt))

	myWMEnt := sdk.Entity{
		ProjectKey:          proj.Key,
		ProjectRepositoryID: repo.ID,
		Type:                sdk.EntityTypeWorkerModel,
		FilePath:            ".cds/worker-models/myworker-model.yml",
		Name:                "myworker-model",
		Branch:              "master",
		Commit:              "123456",
		LastUpdate:          time.Time{},
		Data:                "name: myworkermodel",
	}
	require.NoError(t, entity.Insert(ctx, db, &myWMEnt))

	require.NoError(t, api.craftWorkflowRunV2(ctx, wr.ID))

	wrDB, err := workflow_v2.LoadRunByID(ctx, db, wr.ID)
	require.NoError(t, err)
	require.Equal(t, wrDB.Status, sdk.StatusBuilding)
	wrInfos, err := workflow_v2.LoadRunInfosByRunID(ctx, db, wr.ID)
	require.NoError(t, err)
	require.Equal(t, 0, len(wrInfos))
}

func TestCraftWorkflowRunDepsDifferentRepo(t *testing.T) {
	api, db, _ := newTestAPI(t)
	ctx := context.TODO()

	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))
	admin, _ := assets.InsertAdminUser(t, db)

	vcsProject := assets.InsertTestVCSProject(t, db, proj.ID, "github", "github")
	repo := assets.InsertTestProjectRepository(t, db, vcsProject.ID, "my/repo")

	repoAction1 := assets.InsertTestProjectRepository(t, db, vcsProject.ID, "my/repoAction1")
	repoAction2 := assets.InsertTestProjectRepository(t, db, vcsProject.ID, "my/repoAction2")

	wkName := sdk.RandomString(10)
	wr := sdk.V2WorkflowRun{
		UserID:       admin.ID,
		ProjectKey:   proj.Key,
		Status:       sdk.StatusCrafting,
		VCSServerID:  vcsProject.ID,
		RepositoryID: repo.ID,
		RunNumber:    0,
		RunAttempt:   0,
		WorkflowRef:  "master",
		WorkflowSha:  "123456",
		WorkflowName: wkName,
		WorkflowData: sdk.V2WorkflowRunData{
			Workflow: sdk.V2Workflow{
				Name: wkName,
				Jobs: map[string]sdk.V2Job{
					"job1": {
						Name:        "My super job",
						If:          "cds.workflow == 'toto'",
						Region:      "build",
						WorkerModel: "myworker-model",
						Steps: []sdk.ActionStep{
							{
								ID:   "myfirstStep",
								Uses: fmt.Sprintf("actions/%s/%s/%s/myaction1@master", proj.Key, vcsProject.Name, repoAction1.Name),
							},
							{
								ID:   "mysecondStep",
								Uses: fmt.Sprintf("actions/%s/myaction2", repoAction2.Name),
							},
							{
								ID:   "mythirdStep",
								Uses: fmt.Sprintf("actions/%s/myaction2", repoAction2.Name),
							},
						},
					},
				},
			},
		},
	}
	require.NoError(t, workflow_v2.InsertRun(ctx, db, &wr))

	myactionEnt := sdk.Entity{
		ProjectKey:          proj.Key,
		ProjectRepositoryID: repoAction1.ID,
		Type:                sdk.EntityTypeAction,
		FilePath:            ".cds/actions/myaction.yml",
		Name:                "myaction1",
		Branch:              "master",
		Commit:              "",
		LastUpdate:          time.Time{},
		Data:                "name: myaction",
	}
	require.NoError(t, entity.Insert(ctx, db, &myactionEnt))

	myactionEnt2 := sdk.Entity{
		ProjectKey:          proj.Key,
		ProjectRepositoryID: repoAction2.ID,
		Type:                sdk.EntityTypeAction,
		FilePath:            ".cds/actions/myaction2.yml",
		Name:                "myaction2",
		Branch:              "main",
		Commit:              "",
		LastUpdate:          time.Time{},
		Data:                "name: myaction2",
	}
	require.NoError(t, entity.Insert(ctx, db, &myactionEnt2))

	myWMEnt := sdk.Entity{
		ProjectKey:          proj.Key,
		ProjectRepositoryID: repo.ID,
		Type:                sdk.EntityTypeWorkerModel,
		FilePath:            ".cds/worker-models/myworker-model.yml",
		Name:                "myworker-model",
		Branch:              "master",
		Commit:              "123456",
		LastUpdate:          time.Time{},
		Data:                "name: myworkermodel",
	}
	require.NoError(t, entity.Insert(ctx, db, &myWMEnt))

	// Mock VCS
	s, _ := assets.InsertService(t, db, t.Name()+"_VCS", sdk.TypeVCS)
	// Setup a mock for all services called by the API
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	servicesClients := mock_services.NewMockClient(ctrl)
	services.NewClient = func(_ gorp.SqlExecutor, _ []sdk.Service) services.Client {
		return servicesClients
	}
	defer func() {
		_ = services.Delete(db, s)
		services.NewClient = services.NewDefaultClient
	}()

	// ACTION 2: no branch specified, need to get the default branch. Call only once because for the third action it need to use cache
	servicesClients.EXPECT().
		DoJSONRequest(gomock.Any(), "GET", "/vcs/github/repos/my/repoAction2/branches/?branch=&default=true", gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(
			func(ctx context.Context, method, path string, in interface{}, out interface{}, _ interface{}) (http.Header, int, error) {
				b := &sdk.VCSBranch{
					Default:   true,
					DisplayID: "main",
				}
				*(out.(*sdk.VCSBranch)) = *b
				return nil, 200, nil
			},
		).Times(1)

	require.NoError(t, api.craftWorkflowRunV2(ctx, wr.ID))

	wrDB, err := workflow_v2.LoadRunByID(ctx, db, wr.ID)
	require.NoError(t, err)
	require.Equal(t, wrDB.Status, sdk.StatusBuilding)
	wrInfos, err := workflow_v2.LoadRunInfosByRunID(ctx, db, wr.ID)
	require.NoError(t, err)
	require.Equal(t, 0, len(wrInfos))
}
