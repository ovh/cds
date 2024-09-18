package api

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/ovh/cds/engine/api/entity"
	"github.com/ovh/cds/engine/api/hatchery"
	"github.com/ovh/cds/engine/api/integration"
	"github.com/ovh/cds/engine/api/rbac"
	"github.com/ovh/cds/engine/api/region"
	"github.com/ovh/cds/engine/api/services"
	"github.com/ovh/cds/engine/api/services/mock_services"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/engine/api/workflow_v2"
	"github.com/ovh/cds/sdk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCraftWorkflowRunNoHatchery(t *testing.T) {
	api, db, _ := newTestAPI(t)
	ctx := context.TODO()

	db.Exec("DELETE FROM rbac")
	db.Exec("DELETE FROM region")

	reg := sdk.Region{Name: "build"}
	require.NoError(t, region.Insert(ctx, db, &reg))

	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))
	admin, _ := assets.InsertAdminUser(t, db)

	vcsProject := assets.InsertTestVCSProject(t, db, proj.ID, "github", "github")
	repo := assets.InsertTestProjectRepository(t, db, proj.Key, vcsProject.ID, "my/repo")

	s, _ := assets.InsertService(t, db, t.Name()+"_VCS", sdk.TypeVCS)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	servicesClients := mock_services.NewMockClient(ctrl)
	services.NewClient = func(_ []sdk.Service) services.Client {
		return servicesClients
	}
	t.Cleanup(func() {
		_ = services.Delete(db, s)
		services.NewClient = services.NewDefaultClient
	})
	servicesClients.EXPECT().
		DoJSONRequest(gomock.Any(), "GET", "/vcs/github/repos/my/repo", gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(
			func(ctx context.Context, method, path string, in interface{}, out interface{}, _ interface{}) (http.Header, int, error) {
				b := &sdk.VCSRepo{}
				*(out.(*sdk.VCSRepo)) = *b
				return nil, 200, nil
			},
		).Times(1)

	wkName := sdk.RandomString(10)
	wr := sdk.V2WorkflowRun{
		UserID:       admin.ID,
		ProjectKey:   proj.Key,
		Status:       sdk.V2WorkflowRunStatusCrafting,
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
						Name:   "My super job",
						If:     "cds.workflow == 'toto'",
						Region: "build",
						Steps: []sdk.ActionStep{
							{
								ID: "step1",
							},
						},
					},
				},
			},
		},
		RunEvent: sdk.V2WorkflowRunEvent{
			HookType:  sdk.WorkflowHookTypeRepository,
			Payload:   nil,
			Ref:       "main",
			Sha:       "123456",
			EventName: sdk.WorkflowHookEventNamePush,
		},
	}
	require.NoError(t, workflow_v2.InsertRun(ctx, db, &wr))

	require.NoError(t, api.craftWorkflowRunV2(ctx, wr.ID))

	wrDB, err := workflow_v2.LoadRunByID(ctx, db, wr.ID)
	require.NoError(t, err)
	require.Equal(t, sdk.V2WorkflowRunStatusFail, wrDB.Status)
	wrInfos, err := workflow_v2.LoadRunInfosByRunID(ctx, db, wr.ID)
	require.NoError(t, err)
	require.Equal(t, 1, len(wrInfos))
	require.Equal(t, "wrong configuration on job \"job1\". No hatchery can run it", wrInfos[0].Message)
}

func TestCraftWorkflowRunDepsNotFound(t *testing.T) {
	api, db, _ := newTestAPI(t)
	ctx := context.TODO()

	db.Exec("DELETE FROM rbac")
	db.Exec("DELETE FROM region")

	reg := sdk.Region{Name: "build"}
	require.NoError(t, region.Insert(ctx, db, &reg))

	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))
	admin, _ := assets.InsertAdminUser(t, db)

	vcsProject := assets.InsertTestVCSProject(t, db, proj.ID, "github", "github")
	repo := assets.InsertTestProjectRepository(t, db, proj.Key, vcsProject.ID, "my/repo")

	s, _ := assets.InsertService(t, db, t.Name()+"_VCS", sdk.TypeVCS)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	servicesClients := mock_services.NewMockClient(ctrl)
	services.NewClient = func(_ []sdk.Service) services.Client {
		return servicesClients
	}
	t.Cleanup(func() {
		_ = services.Delete(db, s)
		services.NewClient = services.NewDefaultClient
	})
	servicesClients.EXPECT().
		DoJSONRequest(gomock.Any(), "GET", "/vcs/github/repos/my/repo", gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(
			func(ctx context.Context, method, path string, in interface{}, out interface{}, _ interface{}) (http.Header, int, error) {
				b := &sdk.VCSRepo{}
				*(out.(*sdk.VCSRepo)) = *b
				return nil, 200, nil
			},
		).Times(1)

	wkName := sdk.RandomString(10)
	wr := sdk.V2WorkflowRun{
		UserID:       admin.ID,
		ProjectKey:   proj.Key,
		Status:       sdk.V2WorkflowRunStatusCrafting,
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
						Name:   "My super job",
						If:     "cds.workflow == 'toto'",
						Region: "build",
						RunsOn: sdk.V2JobRunsOn{
							Model: "myworker-model",
						},
						Steps: []sdk.ActionStep{
							{
								ID: "step1",
							},
						},
					},
				},
			},
		},
		RunEvent: sdk.V2WorkflowRunEvent{
			HookType:  sdk.WorkflowHookTypeRepository,
			Payload:   nil,
			Ref:       "main",
			Sha:       "123456",
			EventName: sdk.WorkflowHookEventNamePush,
		},
	}
	require.NoError(t, workflow_v2.InsertRun(ctx, db, &wr))

	// Create hatchery
	hatch := sdk.Hatchery{Name: sdk.RandomString(10), ModelType: ""}
	require.NoError(t, hatchery.Insert(ctx, db, &hatch))

	perm := sdk.RBAC{
		Name: sdk.RandomString(10),
		Hatcheries: []sdk.RBACHatchery{
			{
				RegionID:   reg.ID,
				HatcheryID: hatch.ID,
				Role:       sdk.HatcheryRoleSpawn,
			},
		},
	}
	require.NoError(t, rbac.Insert(ctx, db, &perm))

	require.NoError(t, api.craftWorkflowRunV2(ctx, wr.ID))

	wrDB, err := workflow_v2.LoadRunByID(ctx, db, wr.ID)
	require.NoError(t, err)
	require.Equal(t, sdk.V2WorkflowRunStatusFail, wrDB.Status)
	wrInfos, err := workflow_v2.LoadRunInfosByRunID(ctx, db, wr.ID)
	require.NoError(t, err)
	require.Equal(t, 1, len(wrInfos))
	require.Equal(t, "unable to find workflow dependency: myworker-model", wrInfos[0].Message)
}

func TestCraftWorkflowRunDepsSameRepo(t *testing.T) {
	api, db, _ := newTestAPI(t)
	ctx := context.TODO()

	db.Exec("DELETE FROM rbac")
	db.Exec("DELETE FROM region")

	reg := sdk.Region{Name: "build"}
	require.NoError(t, region.Insert(ctx, db, &reg))

	reg1 := sdk.Region{Name: "myregion"}
	require.NoError(t, region.Insert(ctx, db, &reg1))

	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))
	admin, _ := assets.InsertAdminUser(t, db)

	vcsProject := assets.InsertTestVCSProject(t, db, proj.ID, "github", "github")
	repo := assets.InsertTestProjectRepository(t, db, proj.Key, vcsProject.ID, "my/repo")

	model := sdk.IntegrationModel{Name: sdk.RandomString(10), Event: true, DefaultConfig: sdk.IntegrationConfig{
		"myparam": {
			Value: "myregion",
			Type:  sdk.IntegrationConfigTypeRegion,
		},
	}}
	require.NoError(t, integration.InsertModel(db, &model))
	projInt := sdk.ProjectIntegration{
		Config: sdk.IntegrationConfig{
			"test": sdk.IntegrationConfigValue{
				Description: "here is a test",
				Type:        sdk.IntegrationConfigTypeString,
				Value:       "test",
			},
			"myparam": model.DefaultConfig["myparam"],
		},
		Name:               sdk.RandomString(10),
		ProjectID:          proj.ID,
		Model:              model,
		IntegrationModelID: model.ID,
	}
	require.NoError(t, integration.InsertIntegration(db, &projInt))

	s, _ := assets.InsertService(t, db, t.Name()+"_VCS", sdk.TypeVCS)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	servicesClients := mock_services.NewMockClient(ctrl)
	services.NewClient = func(_ []sdk.Service) services.Client {
		return servicesClients
	}
	t.Cleanup(func() {
		_ = services.Delete(db, s)
		services.NewClient = services.NewDefaultClient
	})
	servicesClients.EXPECT().
		DoJSONRequest(gomock.Any(), "GET", "/vcs/github/repos/my/repo", gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(
			func(ctx context.Context, method, path string, in interface{}, out interface{}, _ interface{}) (http.Header, int, error) {
				b := &sdk.VCSRepo{}
				*(out.(*sdk.VCSRepo)) = *b
				return nil, 200, nil
			},
		).Times(1)

	wkName := sdk.RandomString(10)
	wr := sdk.V2WorkflowRun{
		UserID:       admin.ID,
		ProjectKey:   proj.Key,
		Status:       sdk.V2WorkflowRunStatusCrafting,
		VCSServerID:  vcsProject.ID,
		RepositoryID: repo.ID,
		RunNumber:    0,
		RunAttempt:   0,
		WorkflowRef:  "refs/heads/master",
		WorkflowSha:  "123456",
		WorkflowName: wkName,
		WorkflowData: sdk.V2WorkflowRunData{
			Workflow: sdk.V2Workflow{
				Name: wkName,
				Jobs: map[string]sdk.V2Job{
					"job1": {
						Name: "My super job",
						If:   "cds.workflow == 'toto'",
						RunsOn: sdk.V2JobRunsOn{
							Model: "myworker-model",
						},
						Integrations: []string{projInt.Name},
						Steps: []sdk.ActionStep{
							{
								ID:   "myfirstStep",
								Uses: fmt.Sprintf("actions/%s/%s/%s/myaction", proj.Key, vcsProject.Name, strings.ToUpper(repo.Name)),
							},
							{
								ID:   "mysecondStep",
								Uses: fmt.Sprintf("actions/%s/%s/myaction@refs/heads/master", vcsProject.Name, repo.Name),
							},
							{
								ID:   "mythirdStep",
								Uses: fmt.Sprintf("actions/%s/myaction", repo.Name),
							},
						},
					},
				},
			},
		},
		RunEvent: sdk.V2WorkflowRunEvent{
			HookType:  sdk.WorkflowHookTypeRepository,
			Payload:   nil,
			Ref:       "refs/heads/main",
			Sha:       "123456",
			EventName: sdk.WorkflowHookEventNamePush,
		},
	}
	require.NoError(t, workflow_v2.InsertRun(ctx, db, &wr))

	myactionEnt := sdk.Entity{
		ProjectKey:          proj.Key,
		ProjectRepositoryID: repo.ID,
		Type:                sdk.EntityTypeAction,
		FilePath:            ".cds/actions/myaction.yml",
		Name:                "myaction",
		Ref:                 "refs/heads/master",
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
		Ref:                 "refs/heads/master",
		Commit:              "123456",
		LastUpdate:          time.Time{},
		Data:                "name: myworkermodel",
	}
	require.NoError(t, entity.Insert(ctx, db, &myWMEnt))

	// Create hatchery
	hatch := sdk.Hatchery{Name: sdk.RandomString(10), ModelType: ""}
	require.NoError(t, hatchery.Insert(ctx, db, &hatch))

	perm := sdk.RBAC{
		Name: sdk.RandomString(10),
		Hatcheries: []sdk.RBACHatchery{
			{
				RegionID:   reg1.ID,
				HatcheryID: hatch.ID,
				Role:       sdk.HatcheryRoleSpawn,
			},
		},
	}
	require.NoError(t, rbac.Insert(ctx, db, &perm))

	require.NoError(t, api.craftWorkflowRunV2(ctx, wr.ID))

	wrDB, err := workflow_v2.LoadRunByID(ctx, db, wr.ID)
	require.NoError(t, err)

	runInfos, err := workflow_v2.LoadRunInfosByRunID(ctx, db, wr.ID)
	require.NoError(t, err)
	t.Logf("%+v", runInfos)

	t.Logf("%+v", wrDB.WorkflowData.Actions)
	require.Equal(t, sdk.V2WorkflowRunStatusBuilding, wrDB.Status)
	wrInfos, err := workflow_v2.LoadRunInfosByRunID(ctx, db, wr.ID)
	require.NoError(t, err)
	require.Equal(t, 0, len(wrInfos))

	require.Contains(t, wrDB.WorkflowData.Workflow.Jobs["job1"].RunsOn.Model, "myworker-model@refs/heads/master")
	require.Contains(t, wrDB.WorkflowData.Workflow.Jobs["job1"].Region, "myregion")
}

func TestCraftWorkflowRunDepsDifferentRepo(t *testing.T) {
	api, db, _ := newTestAPI(t)
	ctx := context.TODO()

	db.Exec("DELETE FROM rbac")
	db.Exec("DELETE FROM region")

	reg := sdk.Region{Name: "build"}
	require.NoError(t, region.Insert(ctx, db, &reg))

	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))
	admin, _ := assets.InsertAdminUser(t, db)

	vcsProject := assets.InsertTestVCSProject(t, db, proj.ID, "github", "github")
	repo := assets.InsertTestProjectRepository(t, db, proj.Key, vcsProject.ID, "my/repo")

	repoAction1 := assets.InsertTestProjectRepository(t, db, proj.Key, vcsProject.ID, "my/repoAction1")
	repoAction2 := assets.InsertTestProjectRepository(t, db, proj.Key, vcsProject.ID, "my/repoAction2")

	wkName := sdk.RandomString(10)
	wr := sdk.V2WorkflowRun{
		UserID:       admin.ID,
		ProjectKey:   proj.Key,
		Status:       sdk.V2WorkflowRunStatusCrafting,
		VCSServerID:  vcsProject.ID,
		RepositoryID: repo.ID,
		RunNumber:    0,
		RunAttempt:   0,
		WorkflowRef:  "refs/heads/master",
		WorkflowSha:  "123456",
		WorkflowName: wkName,
		WorkflowData: sdk.V2WorkflowRunData{
			Workflow: sdk.V2Workflow{
				Name: wkName,
				Jobs: map[string]sdk.V2Job{
					"job1": {
						Name:   "My super job",
						If:     "cds.workflow == 'toto'",
						Region: "build",
						RunsOn: sdk.V2JobRunsOn{
							Model: "myworker-model",
						},
						Steps: []sdk.ActionStep{
							{
								ID:   "myfirstStep",
								Uses: fmt.Sprintf("actions/%s/%s/%s/myaction1@refs/heads/master", proj.Key, vcsProject.Name, repoAction1.Name),
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
		RunEvent: sdk.V2WorkflowRunEvent{
			HookType:  sdk.WorkflowHookTypeRepository,
			Payload:   nil,
			Ref:       "refs/heads/main",
			Sha:       "123456",
			EventName: sdk.WorkflowHookEventNamePush,
		},
	}
	require.NoError(t, workflow_v2.InsertRun(ctx, db, &wr))

	myactionEnt := sdk.Entity{
		ProjectKey:          proj.Key,
		ProjectRepositoryID: repoAction1.ID,
		Type:                sdk.EntityTypeAction,
		FilePath:            ".cds/actions/myaction.yml",
		Name:                "myaction1",
		Ref:                 "refs/heads/master",
		Commit:              "HEAD",
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
		Ref:                 "refs/heads/main",
		Commit:              "HEAD",
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
		Ref:                 "refs/heads/master",
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
	services.NewClient = func(_ []sdk.Service) services.Client {
		return servicesClients
	}
	defer func() {
		_ = services.Delete(db, s)
		services.NewClient = services.NewDefaultClient
	}()

	// ACTION 2: no branch specified, need to get the default branch. Call only once because for the third action it need to use cache
	servicesClients.EXPECT().
		DoJSONRequest(gomock.Any(), "GET", "/vcs/github/repos/my/repoaction2/branches/?branch=&default=true", gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(
			func(ctx context.Context, method, path string, in interface{}, out interface{}, _ interface{}) (http.Header, int, error) {
				b := &sdk.VCSBranch{
					Default:   true,
					DisplayID: "main",
					ID:        "refs/heads/main",
				}
				*(out.(*sdk.VCSBranch)) = *b
				return nil, 200, nil
			},
		).Times(1)
	servicesClients.EXPECT().
		DoJSONRequest(gomock.Any(), "GET", "/vcs/github/repos/my/repo", gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(
			func(ctx context.Context, method, path string, in interface{}, out interface{}, _ interface{}) (http.Header, int, error) {
				b := &sdk.VCSRepo{}
				*(out.(*sdk.VCSRepo)) = *b
				return nil, 200, nil
			},
		).Times(1)

	// Create hatchery
	hatch := sdk.Hatchery{Name: sdk.RandomString(10), ModelType: ""}
	require.NoError(t, hatchery.Insert(ctx, db, &hatch))

	require.NoError(t, rbac.Insert(ctx, db, &sdk.RBAC{
		Name: sdk.RandomString(10),
		Hatcheries: []sdk.RBACHatchery{
			{
				RegionID:   reg.ID,
				HatcheryID: hatch.ID,
				Role:       sdk.HatcheryRoleSpawn,
			},
		},
	}))

	require.NoError(t, api.craftWorkflowRunV2(ctx, wr.ID))

	wrDB, err := workflow_v2.LoadRunByID(ctx, db, wr.ID)
	require.NoError(t, err)
	assert.Equal(t, wrDB.Status, sdk.V2WorkflowRunStatusBuilding)
	wrInfos, err := workflow_v2.LoadRunInfosByRunID(ctx, db, wr.ID)
	require.NoError(t, err)
	require.Equal(t, 0, len(wrInfos), "Error found: %v", wrInfos)
}
