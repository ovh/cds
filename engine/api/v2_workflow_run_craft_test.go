package api

import (
	"context"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"testing"
	"time"

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
	"go.uber.org/mock/gomock"
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
		DeprecatedUserID: admin.ID,
		ProjectKey:       proj.Key,
		Status:           sdk.V2WorkflowRunStatusCrafting,
		VCSServerID:      vcsProject.ID,
		RepositoryID:     repo.ID,
		RunNumber:        0,
		RunAttempt:       0,
		WorkflowRef:      "master",
		WorkflowSha:      "123456789",
		WorkflowName:     wkName,
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
								ID:  "step1",
								Run: "echo toto",
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
			Sha:       "123456789",
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
	t.Logf("%+v", wrInfos)
	require.Equal(t, 1, len(wrInfos))
	require.Equal(t, "wrong configuration on job \"job1\". No hatchery can run it with model []", wrInfos[0].Message)
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
		DeprecatedUserID: admin.ID,
		ProjectKey:       proj.Key,
		Status:           sdk.V2WorkflowRunStatusCrafting,
		VCSServerID:      vcsProject.ID,
		RepositoryID:     repo.ID,
		RunNumber:        0,
		RunAttempt:       0,
		WorkflowRef:      "master",
		WorkflowSha:      "123456789",
		WorkflowName:     wkName,
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
								ID:  "step1",
								Run: "echo toto",
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
			Sha:       "123456789",
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
		DeprecatedUserID: admin.ID,
		ProjectKey:       proj.Key,
		Status:           sdk.V2WorkflowRunStatusCrafting,
		VCSServerID:      vcsProject.ID,
		RepositoryID:     repo.ID,
		RunNumber:        0,
		RunAttempt:       0,
		WorkflowRef:      "refs/heads/master",
		WorkflowSha:      "123456789",
		WorkflowName:     wkName,
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
			Sha:       "123456789",
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
		Commit:              "123456789",
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
		Commit:              "123456789",
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
		DeprecatedUserID: admin.ID,
		ProjectKey:       proj.Key,
		Status:           sdk.V2WorkflowRunStatusCrafting,
		VCSServerID:      vcsProject.ID,
		RepositoryID:     repo.ID,
		RunNumber:        0,
		RunAttempt:       0,
		WorkflowRef:      "refs/heads/master",
		WorkflowSha:      "123456789",
		WorkflowName:     wkName,
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
			Sha:       "123456789",
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
		Commit:              "123456789",
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

func TestCraftWorkflowRunCustomVersion_Cargo(t *testing.T) {
	api, db, _ := newTestAPI(t)
	ctx := context.TODO()

	db.Exec("DELETE FROM rbac")
	db.Exec("DELETE FROM region")

	reg := sdk.Region{Name: "build"}
	require.NoError(t, region.Insert(ctx, db, &reg))

	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))
	admin, _ := assets.InsertAdminUser(t, db)

	vcsProject := assets.InsertTestVCSProject(t, db, proj.ID, "bitbucketserver", "bitbucketserver")
	repo := assets.InsertTestProjectRepository(t, db, proj.Key, vcsProject.ID, "my/repo")

	wkName := sdk.RandomString(10)
	wr := sdk.V2WorkflowRun{
		DeprecatedUserID: admin.ID,
		ProjectKey:       proj.Key,
		Status:           sdk.V2WorkflowRunStatusCrafting,
		VCSServerID:      vcsProject.ID,
		RepositoryID:     repo.ID,
		RunNumber:        1,
		RunAttempt:       0,
		WorkflowRef:      "refs/heads/master",
		WorkflowSha:      "123456789",
		WorkflowName:     wkName,
		WorkflowData: sdk.V2WorkflowRunData{
			Workflow: sdk.V2Workflow{
				Name: wkName,
				Semver: &sdk.WorkflowSemver{
					From: "cargo",
					Path: "Cargo.toml",
				},
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
								Run: "echo toto",
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
			Sha:       "123456789",
			EventName: sdk.WorkflowHookEventNamePush,
		},
	}
	require.NoError(t, workflow_v2.InsertRun(ctx, db, &wr))

	myWMEnt := sdk.Entity{
		ProjectKey:          proj.Key,
		ProjectRepositoryID: repo.ID,
		Type:                sdk.EntityTypeWorkerModel,
		FilePath:            ".cds/worker-models/myworker-model.yml",
		Name:                "myworker-model",
		Ref:                 "refs/heads/master",
		Commit:              "123456789",
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

	servicesClients.EXPECT().
		DoJSONRequest(gomock.Any(), "GET", "/vcs/bitbucketserver/repos/my/repo/content/Cargo.toml?commit=123456789", gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, method, path string, in interface{}, out interface{}, _ interface{}) (http.Header, int, error) {
			b := &sdk.VCSContent{
				IsFile: true,
				Content: `[package]
name = "mycargo"
version = "0.85.0"`,
			}
			*(out.(*sdk.VCSContent)) = *b
			return nil, 200, nil
		}).Times(2)

	servicesClients.EXPECT().
		DoJSONRequest(gomock.Any(), "GET", "/vcs/bitbucketserver/repos/my/repo/branches/?branch=&default=true", gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, method, path string, in interface{}, out interface{}, _ interface{}) (http.Header, int, error) {
			b := &sdk.VCSBranch{
				DisplayID: "main",
				ID:        "refs/heads/main",
				Default:   true,
			}
			*(out.(*sdk.VCSBranch)) = *b
			return nil, 200, nil
		}).Times(2)

	servicesClients.EXPECT().
		DoJSONRequest(gomock.Any(), "GET", "/vcs/bitbucketserver/repos/my/repo", gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(
			func(ctx context.Context, method, path string, in interface{}, out interface{}, _ interface{}) (http.Header, int, error) {
				b := &sdk.VCSRepo{}
				*(out.(*sdk.VCSRepo)) = *b
				return nil, 200, nil
			},
		).Times(2)

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

	require.Equal(t, "0.85.0", wrDB.Contexts.CDS.Version)

	version, err := workflow_v2.LoadWorkflowVersion(ctx, db, wrDB.Contexts.CDS.ProjectKey, wrDB.Contexts.CDS.WorkflowVCSServer, wrDB.Contexts.CDS.WorkflowRepository, wrDB.Contexts.CDS.Workflow, "0.85.0")
	require.NoError(t, err)

	require.NotNil(t, version)

	// Update the run and craft it again
	wrDB.Status = sdk.V2WorkflowRunStatusCrafting
	require.NoError(t, workflow_v2.UpdateRun(ctx, db, wrDB))
	require.NoError(t, api.craftWorkflowRunV2(ctx, wrDB.ID))

	wrDB, err = workflow_v2.LoadRunByID(ctx, db, wr.ID)
	require.NoError(t, err)
	versions, err := workflow_v2.LoadAllVerionsByWorkflow(ctx, db, wrDB.Contexts.CDS.ProjectKey, wrDB.Contexts.CDS.WorkflowVCSServer, wrDB.Contexts.CDS.WorkflowRepository, wrDB.Contexts.CDS.Workflow)
	require.NoError(t, err)
	require.Equal(t, 1, len(versions))
	require.Equal(t, "0.85.0-1.sha."+wrDB.Contexts.Git.ShaShort, wrDB.Contexts.CDS.Version)
}

func TestCraftWorkflowRunCustomVersion_Helm(t *testing.T) {
	api, db, _ := newTestAPI(t)
	ctx := context.TODO()

	db.Exec("DELETE FROM rbac")
	db.Exec("DELETE FROM region")

	reg := sdk.Region{Name: "build"}
	require.NoError(t, region.Insert(ctx, db, &reg))

	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))
	admin, _ := assets.InsertAdminUser(t, db)

	vcsProject := assets.InsertTestVCSProject(t, db, proj.ID, "bitbucketserver", "bitbucketserver")
	repo := assets.InsertTestProjectRepository(t, db, proj.Key, vcsProject.ID, "my/repo")

	wkName := sdk.RandomString(10)
	wr := sdk.V2WorkflowRun{
		DeprecatedUserID: admin.ID,
		ProjectKey:       proj.Key,
		Status:           sdk.V2WorkflowRunStatusCrafting,
		VCSServerID:      vcsProject.ID,
		RepositoryID:     repo.ID,
		RunNumber:        1,
		RunAttempt:       0,
		WorkflowRef:      "refs/heads/master",
		WorkflowSha:      "123456789",
		WorkflowName:     wkName,
		WorkflowData: sdk.V2WorkflowRunData{
			Workflow: sdk.V2Workflow{
				Name: wkName,
				Semver: &sdk.WorkflowSemver{
					From:        "helm",
					Path:        "Chart.yaml",
					ReleaseRefs: []string{"refs/heads/mai*"},
				},
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
								Run: "echo toto",
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
			Sha:       "123456789",
			EventName: sdk.WorkflowHookEventNamePush,
		},
	}
	require.NoError(t, workflow_v2.InsertRun(ctx, db, &wr))

	myWMEnt := sdk.Entity{
		ProjectKey:          proj.Key,
		ProjectRepositoryID: repo.ID,
		Type:                sdk.EntityTypeWorkerModel,
		FilePath:            ".cds/worker-models/myworker-model.yml",
		Name:                "myworker-model",
		Ref:                 "refs/heads/master",
		Commit:              "123456789",
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

	servicesClients.EXPECT().
		DoJSONRequest(gomock.Any(), "GET", "/vcs/bitbucketserver/repos/my/repo/content/Chart.yaml?commit=123456789", gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, method, path string, in interface{}, out interface{}, _ interface{}) (http.Header, int, error) {
			b := &sdk.VCSContent{
				IsFile: true,
				Content: `apiVersion: 1.0
name: chartName
version: 1.11.0
kubeVersion: 1.19.0
description: A single-sentence description of this project`,
			}
			*(out.(*sdk.VCSContent)) = *b
			return nil, 200, nil
		}).Times(2)

	servicesClients.EXPECT().
		DoJSONRequest(gomock.Any(), "GET", "/vcs/bitbucketserver/repos/my/repo", gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(
			func(ctx context.Context, method, path string, in interface{}, out interface{}, _ interface{}) (http.Header, int, error) {
				b := &sdk.VCSRepo{}
				*(out.(*sdk.VCSRepo)) = *b
				return nil, 200, nil
			},
		).Times(2)

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

	require.Equal(t, "1.11.0", wrDB.Contexts.CDS.Version)

	version, err := workflow_v2.LoadWorkflowVersion(ctx, db, wrDB.Contexts.CDS.ProjectKey, wrDB.Contexts.CDS.WorkflowVCSServer, wrDB.Contexts.CDS.WorkflowRepository, wrDB.Contexts.CDS.Workflow, "1.11.0")
	require.NoError(t, err)

	require.NotNil(t, version)

	// Update the run and craft it again
	wrDB.Status = sdk.V2WorkflowRunStatusCrafting
	require.NoError(t, workflow_v2.UpdateRun(ctx, db, wrDB))
	require.NoError(t, api.craftWorkflowRunV2(ctx, wrDB.ID))

	wrDB, err = workflow_v2.LoadRunByID(ctx, db, wr.ID)
	require.NoError(t, err)
	versions, err := workflow_v2.LoadAllVerionsByWorkflow(ctx, db, wrDB.Contexts.CDS.ProjectKey, wrDB.Contexts.CDS.WorkflowVCSServer, wrDB.Contexts.CDS.WorkflowRepository, wrDB.Contexts.CDS.Workflow)
	require.NoError(t, err)
	require.Equal(t, 1, len(versions))
	require.Equal(t, "1.11.0-1.sha."+wrDB.Contexts.Git.ShaShort, wrDB.Contexts.CDS.Version)
}

func TestCraftWorkflowRunCustomVersion_GitOnBranch(t *testing.T) {
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

	wkName := sdk.RandomString(10)
	wr := sdk.V2WorkflowRun{
		DeprecatedUserID: admin.ID,
		ProjectKey:       proj.Key,
		Status:           sdk.V2WorkflowRunStatusCrafting,
		VCSServerID:      vcsProject.ID,
		RepositoryID:     repo.ID,
		RunNumber:        1,
		RunAttempt:       0,
		WorkflowRef:      "refs/heads/develop",
		WorkflowSha:      "123456789",
		WorkflowName:     wkName,
		WorkflowData: sdk.V2WorkflowRunData{
			Workflow: sdk.V2Workflow{
				Name: wkName,
				Semver: &sdk.WorkflowSemver{
					From: "git",
					Schema: map[string]string{
						"refs/heads/develop": "${{git.version}}-rc-${{git.sha_short}}",
					},
				},
				Jobs: map[string]sdk.V2Job{
					"job1": {
						Name:   "My super job",
						If:     "cds.workflow == 'toto'",
						Region: "build",
						RunsOn: sdk.V2JobRunsOn{
							Model: "myworker-model-${{ xx }}",
						},
						Steps: []sdk.ActionStep{
							{
								Run: "echo toto",
							},
						},
					},
				},
			},
		},
		RunEvent: sdk.V2WorkflowRunEvent{
			HookType:  sdk.WorkflowHookTypeRepository,
			Payload:   nil,
			Ref:       "refs/heads/develop",
			Sha:       "123456789",
			EventName: sdk.WorkflowHookEventNamePush,
		},
	}
	require.NoError(t, workflow_v2.InsertRun(ctx, db, &wr))

	myWMEnt := sdk.Entity{
		ProjectKey:          proj.Key,
		ProjectRepositoryID: repo.ID,
		Type:                sdk.EntityTypeWorkerModel,
		FilePath:            ".cds/worker-models/myworker-model.yml",
		Name:                "myworker-model",
		Ref:                 "refs/heads/master",
		Commit:              "123456789",
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

	require.Equal(t, "0.1.0-rc-1234567", wrDB.Contexts.CDS.Version)
}

func TestCraftWorkflowRunCustomVersion_GitOnTag(t *testing.T) {
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

	wkName := sdk.RandomString(10)
	wr := sdk.V2WorkflowRun{
		DeprecatedUserID: admin.ID,
		ProjectKey:       proj.Key,
		Status:           sdk.V2WorkflowRunStatusCrafting,
		VCSServerID:      vcsProject.ID,
		RepositoryID:     repo.ID,
		RunNumber:        1,
		RunAttempt:       0,
		WorkflowRef:      "refs/tags/1.0.0",
		WorkflowSha:      "123456789",
		WorkflowName:     wkName,
		WorkflowData: sdk.V2WorkflowRunData{
			Workflow: sdk.V2Workflow{
				Name: wkName,
				Semver: &sdk.WorkflowSemver{
					From: "git",
					Schema: map[string]string{
						"refs/heads/develop": "${{git.version}}-rc-${{git.sha_short}}",
					},
				},
				Jobs: map[string]sdk.V2Job{
					"job1": {
						Name:   "My super job",
						If:     "cds.workflow == 'toto'",
						Region: "build",
						RunsOn: sdk.V2JobRunsOn{
							Model: "myworker-model-${{ xx }}",
						},
						Steps: []sdk.ActionStep{
							{
								Run: "echo toto",
							},
						},
					},
				},
			},
		},
		RunEvent: sdk.V2WorkflowRunEvent{
			HookType:      sdk.WorkflowHookTypeRepository,
			Payload:       nil,
			Ref:           "refs/tags/1.0.0",
			Sha:           "123456789",
			EventName:     sdk.WorkflowHookEventNamePush,
			SemverCurrent: "1.0.0",
		},
	}
	require.NoError(t, workflow_v2.InsertRun(ctx, db, &wr))

	myWMEnt := sdk.Entity{
		ProjectKey:          proj.Key,
		ProjectRepositoryID: repo.ID,
		Type:                sdk.EntityTypeWorkerModel,
		FilePath:            ".cds/worker-models/myworker-model.yml",
		Name:                "myworker-model",
		Ref:                 "refs/tags/1.0.0",
		Commit:              "123456789",
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

	require.Equal(t, "1.0.0", wrDB.Contexts.CDS.Version)
}

func TestCraftWorkflowRunCustomVersion_NpmYarn(t *testing.T) {
	api, db, _ := newTestAPI(t)
	ctx := context.TODO()

	db.Exec("DELETE FROM rbac")
	db.Exec("DELETE FROM region")

	reg := sdk.Region{Name: "build"}
	require.NoError(t, region.Insert(ctx, db, &reg))

	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))
	admin, _ := assets.InsertAdminUser(t, db)

	vcsProject := assets.InsertTestVCSProject(t, db, proj.ID, "bitbucketserver", "bitbucketserver")
	repo := assets.InsertTestProjectRepository(t, db, proj.Key, vcsProject.ID, "my/repo")

	wkName := sdk.RandomString(10)
	wr := sdk.V2WorkflowRun{
		DeprecatedUserID: admin.ID,
		ProjectKey:       proj.Key,
		Status:           sdk.V2WorkflowRunStatusCrafting,
		VCSServerID:      vcsProject.ID,
		RepositoryID:     repo.ID,
		RunNumber:        1,
		RunAttempt:       0,
		WorkflowRef:      "refs/heads/master",
		WorkflowSha:      "123456789",
		WorkflowName:     wkName,
		WorkflowData: sdk.V2WorkflowRunData{
			Workflow: sdk.V2Workflow{
				Name: wkName,
				Semver: &sdk.WorkflowSemver{
					From:        "yarn",
					Path:        "package.json",
					ReleaseRefs: []string{"refs/heads/mai*"},
				},
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
								Run: "echo toto",
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
			Sha:       "123456789",
			EventName: sdk.WorkflowHookEventNamePush,
		},
	}
	require.NoError(t, workflow_v2.InsertRun(ctx, db, &wr))

	myWMEnt := sdk.Entity{
		ProjectKey:          proj.Key,
		ProjectRepositoryID: repo.ID,
		Type:                sdk.EntityTypeWorkerModel,
		FilePath:            ".cds/worker-models/myworker-model.yml",
		Name:                "myworker-model",
		Ref:                 "refs/heads/master",
		Commit:              "123456789",
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

	servicesClients.EXPECT().
		DoJSONRequest(gomock.Any(), "GET", "/vcs/bitbucketserver/repos/my/repo/content/package.json?commit=123456789", gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, method, path string, in interface{}, out interface{}, _ interface{}) (http.Header, int, error) {
			b := &sdk.VCSContent{
				IsFile: true,
				Content: `{
  "name": "blabla",
  "version": "1.2.3"}`,
			}
			*(out.(*sdk.VCSContent)) = *b
			return nil, 200, nil
		}).Times(2)

	servicesClients.EXPECT().
		DoJSONRequest(gomock.Any(), "GET", "/vcs/bitbucketserver/repos/my/repo", gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(
			func(ctx context.Context, method, path string, in interface{}, out interface{}, _ interface{}) (http.Header, int, error) {
				b := &sdk.VCSRepo{}
				*(out.(*sdk.VCSRepo)) = *b
				return nil, 200, nil
			},
		).Times(2)

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

	require.Equal(t, "1.2.3", wrDB.Contexts.CDS.Version)

	version, err := workflow_v2.LoadWorkflowVersion(ctx, db, wrDB.Contexts.CDS.ProjectKey, wrDB.Contexts.CDS.WorkflowVCSServer, wrDB.Contexts.CDS.WorkflowRepository, wrDB.Contexts.CDS.Workflow, "1.2.3")
	require.NoError(t, err)

	require.NotNil(t, version)

	// Update the run and craft it again
	wrDB.Status = sdk.V2WorkflowRunStatusCrafting
	require.NoError(t, workflow_v2.UpdateRun(ctx, db, wrDB))
	require.NoError(t, api.craftWorkflowRunV2(ctx, wrDB.ID))

	wrDB, err = workflow_v2.LoadRunByID(ctx, db, wr.ID)
	require.NoError(t, err)
	versions, err := workflow_v2.LoadAllVerionsByWorkflow(ctx, db, wrDB.Contexts.CDS.ProjectKey, wrDB.Contexts.CDS.WorkflowVCSServer, wrDB.Contexts.CDS.WorkflowRepository, wrDB.Contexts.CDS.Workflow)
	require.NoError(t, err)
	require.Equal(t, 1, len(versions))
	require.Equal(t, "1.2.3-1.sha."+wrDB.Contexts.Git.ShaShort, wrDB.Contexts.CDS.Version)

}

func TestCraftWorkflowRunCustomVersion_File(t *testing.T) {
	api, db, _ := newTestAPI(t)
	ctx := context.TODO()

	db.Exec("DELETE FROM rbac")
	db.Exec("DELETE FROM region")

	reg := sdk.Region{Name: "build"}
	require.NoError(t, region.Insert(ctx, db, &reg))

	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))
	admin, _ := assets.InsertAdminUser(t, db)

	vcsProject := assets.InsertTestVCSProject(t, db, proj.ID, "bitbucketserver", "bitbucketserver")
	repo := assets.InsertTestProjectRepository(t, db, proj.Key, vcsProject.ID, "my/repo")

	wkName := sdk.RandomString(10)
	wr := sdk.V2WorkflowRun{
		DeprecatedUserID: admin.ID,
		ProjectKey:       proj.Key,
		Status:           sdk.V2WorkflowRunStatusCrafting,
		VCSServerID:      vcsProject.ID,
		RepositoryID:     repo.ID,
		RunNumber:        1,
		RunAttempt:       0,
		WorkflowRef:      "refs/heads/master",
		WorkflowSha:      "123456789",
		WorkflowName:     wkName,
		WorkflowData: sdk.V2WorkflowRunData{
			Workflow: sdk.V2Workflow{
				Name: wkName,
				Semver: &sdk.WorkflowSemver{
					From:        "file",
					Path:        ".version",
					ReleaseRefs: []string{"refs/heads/mai*"},
				},
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
								Run: "echo toto",
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
			Sha:       "123456789",
			EventName: sdk.WorkflowHookEventNamePush,
		},
	}
	require.NoError(t, workflow_v2.InsertRun(ctx, db, &wr))

	myWMEnt := sdk.Entity{
		ProjectKey:          proj.Key,
		ProjectRepositoryID: repo.ID,
		Type:                sdk.EntityTypeWorkerModel,
		FilePath:            ".cds/worker-models/myworker-model.yml",
		Name:                "myworker-model",
		Ref:                 "refs/heads/master",
		Commit:              "123456789",
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

	servicesClients.EXPECT().
		DoJSONRequest(gomock.Any(), "GET", "/vcs/bitbucketserver/repos/my/repo/content/.version?commit=123456789", gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, method, path string, in interface{}, out interface{}, _ interface{}) (http.Header, int, error) {
			b := &sdk.VCSContent{
				IsFile:  true,
				Content: `6.6.6`,
			}
			*(out.(*sdk.VCSContent)) = *b
			return nil, 200, nil
		}).Times(2)

	servicesClients.EXPECT().
		DoJSONRequest(gomock.Any(), "GET", "/vcs/bitbucketserver/repos/my/repo", gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(
			func(ctx context.Context, method, path string, in interface{}, out interface{}, _ interface{}) (http.Header, int, error) {
				b := &sdk.VCSRepo{}
				*(out.(*sdk.VCSRepo)) = *b
				return nil, 200, nil
			},
		).Times(2)

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

	require.Equal(t, "6.6.6", wrDB.Contexts.CDS.Version)

	version, err := workflow_v2.LoadWorkflowVersion(ctx, db, wrDB.Contexts.CDS.ProjectKey, wrDB.Contexts.CDS.WorkflowVCSServer, wrDB.Contexts.CDS.WorkflowRepository, wrDB.Contexts.CDS.Workflow, "6.6.6")
	require.NoError(t, err)

	require.NotNil(t, version)

	// Update the run and craft it again
	wrDB.Status = sdk.V2WorkflowRunStatusCrafting
	require.NoError(t, workflow_v2.UpdateRun(ctx, db, wrDB))
	require.NoError(t, api.craftWorkflowRunV2(ctx, wrDB.ID))

	wrDB, err = workflow_v2.LoadRunByID(ctx, db, wr.ID)
	require.NoError(t, err)
	versions, err := workflow_v2.LoadAllVerionsByWorkflow(ctx, db, wrDB.Contexts.CDS.ProjectKey, wrDB.Contexts.CDS.WorkflowVCSServer, wrDB.Contexts.CDS.WorkflowRepository, wrDB.Contexts.CDS.Workflow)
	require.NoError(t, err)
	require.Equal(t, 1, len(versions))
	require.Equal(t, "6.6.6-1.sha."+wrDB.Contexts.Git.ShaShort, wrDB.Contexts.CDS.Version)
}

func TestCraftWorkflowRunCustomVersion_Poetry(t *testing.T) {
	api, db, _ := newTestAPI(t)
	ctx := context.TODO()

	db.Exec("DELETE FROM rbac")
	db.Exec("DELETE FROM region")

	reg := sdk.Region{Name: "build"}
	require.NoError(t, region.Insert(ctx, db, &reg))

	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))
	admin, _ := assets.InsertAdminUser(t, db)

	vcsProject := assets.InsertTestVCSProject(t, db, proj.ID, "bitbucketserver", "bitbucketserver")
	repo := assets.InsertTestProjectRepository(t, db, proj.Key, vcsProject.ID, "my/repo")

	wkName := sdk.RandomString(10)
	wr := sdk.V2WorkflowRun{
		DeprecatedUserID: admin.ID,
		ProjectKey:       proj.Key,
		Status:           sdk.V2WorkflowRunStatusCrafting,
		VCSServerID:      vcsProject.ID,
		RepositoryID:     repo.ID,
		RunNumber:        1,
		RunAttempt:       0,
		WorkflowRef:      "refs/heads/master",
		WorkflowSha:      "123456789",
		WorkflowName:     wkName,
		WorkflowData: sdk.V2WorkflowRunData{
			Workflow: sdk.V2Workflow{
				Name: wkName,
				Semver: &sdk.WorkflowSemver{
					From:        "poetry",
					Path:        "pyproject.toml",
					ReleaseRefs: []string{"refs/heads/mai*"},
				},
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
								Run: "echo toto",
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
			Sha:       "123456789",
			EventName: sdk.WorkflowHookEventNamePush,
		},
	}
	require.NoError(t, workflow_v2.InsertRun(ctx, db, &wr))

	myWMEnt := sdk.Entity{
		ProjectKey:          proj.Key,
		ProjectRepositoryID: repo.ID,
		Type:                sdk.EntityTypeWorkerModel,
		FilePath:            ".cds/worker-models/myworker-model.yml",
		Name:                "myworker-model",
		Ref:                 "refs/heads/master",
		Commit:              "123456789",
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

	servicesClients.EXPECT().
		DoJSONRequest(gomock.Any(), "GET", "/vcs/bitbucketserver/repos/my/repo/content/pyproject.toml?commit=123456789", gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, method, path string, in interface{}, out interface{}, _ interface{}) (http.Header, int, error) {
			b := &sdk.VCSContent{
				IsFile: true,
				Content: `[tool.poetry]
name = "poetry"
version = "2.0.0"
description = "Python dependency management and packaging made easy."
authors = []
maintainers = []
license = "MIT"
readme = "README.md"`,
			}
			*(out.(*sdk.VCSContent)) = *b
			return nil, 200, nil
		}).Times(2)

	servicesClients.EXPECT().
		DoJSONRequest(gomock.Any(), "GET", "/vcs/bitbucketserver/repos/my/repo", gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(
			func(ctx context.Context, method, path string, in interface{}, out interface{}, _ interface{}) (http.Header, int, error) {
				b := &sdk.VCSRepo{}
				*(out.(*sdk.VCSRepo)) = *b
				return nil, 200, nil
			},
		).Times(2)

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

	require.Equal(t, "2.0.0", wrDB.Contexts.CDS.Version)

	version, err := workflow_v2.LoadWorkflowVersion(ctx, db, wrDB.Contexts.CDS.ProjectKey, wrDB.Contexts.CDS.WorkflowVCSServer, wrDB.Contexts.CDS.WorkflowRepository, wrDB.Contexts.CDS.Workflow, "2.0.0")
	require.NoError(t, err)

	require.NotNil(t, version)

	// Update the run and craft it again
	wrDB.Status = sdk.V2WorkflowRunStatusCrafting
	require.NoError(t, workflow_v2.UpdateRun(ctx, db, wrDB))
	require.NoError(t, api.craftWorkflowRunV2(ctx, wrDB.ID))

	wrDB, err = workflow_v2.LoadRunByID(ctx, db, wr.ID)
	require.NoError(t, err)
	versions, err := workflow_v2.LoadAllVerionsByWorkflow(ctx, db, wrDB.Contexts.CDS.ProjectKey, wrDB.Contexts.CDS.WorkflowVCSServer, wrDB.Contexts.CDS.WorkflowRepository, wrDB.Contexts.CDS.Workflow)
	require.NoError(t, err)
	require.Equal(t, 1, len(versions))
	require.Equal(t, "2.0.0-1.sha."+wrDB.Contexts.Git.ShaShort, wrDB.Contexts.CDS.Version)
}

func TestCraftWorkflowRunCustomVersion_Debian(t *testing.T) {
	api, db, _ := newTestAPI(t)
	ctx := context.TODO()

	db.Exec("DELETE FROM rbac")
	db.Exec("DELETE FROM region")

	reg := sdk.Region{Name: "build"}
	require.NoError(t, region.Insert(ctx, db, &reg))

	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))
	admin, _ := assets.InsertAdminUser(t, db)

	vcsProject := assets.InsertTestVCSProject(t, db, proj.ID, "bitbucketserver", "bitbucketserver")
	repo := assets.InsertTestProjectRepository(t, db, proj.Key, vcsProject.ID, "my/repo")

	wkName := sdk.RandomString(10)
	wr := sdk.V2WorkflowRun{
		DeprecatedUserID: admin.ID,
		ProjectKey:       proj.Key,
		Status:           sdk.V2WorkflowRunStatusCrafting,
		VCSServerID:      vcsProject.ID,
		RepositoryID:     repo.ID,
		RunNumber:        1,
		RunAttempt:       0,
		WorkflowRef:      "refs/heads/master",
		WorkflowSha:      "123456789",
		WorkflowName:     wkName,
		WorkflowData: sdk.V2WorkflowRunData{
			Workflow: sdk.V2Workflow{
				Name: wkName,
				Semver: &sdk.WorkflowSemver{
					From:        "debian",
					Path:        "debian/changelog",
					ReleaseRefs: []string{"refs/heads/mai*"},
				},
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
								Run: "echo toto",
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
			Sha:       "123456789",
			EventName: sdk.WorkflowHookEventNamePush,
		},
	}
	require.NoError(t, workflow_v2.InsertRun(ctx, db, &wr))

	myWMEnt := sdk.Entity{
		ProjectKey:          proj.Key,
		ProjectRepositoryID: repo.ID,
		Type:                sdk.EntityTypeWorkerModel,
		FilePath:            ".cds/worker-models/myworker-model.yml",
		Name:                "myworker-model",
		Ref:                 "refs/heads/master",
		Commit:              "123456789",
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

	servicesClients.EXPECT().
		DoJSONRequest(gomock.Any(), "GET", "/vcs/bitbucketserver/repos/my/repo/content/debian%2Fchangelog?commit=123456789", gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, method, path string, in interface{}, out interface{}, _ interface{}) (http.Header, int, error) {
			b := &sdk.VCSContent{
				IsFile: true,
				Content: `mypackage (0.9.12-1) UNRELEASED; urgency=low
2
3   * Initial Release. Closes: #12345
4   * This is my first Debian package.
5   * Adjusted the Makefile to fix $(DESTDIR) problems.
6
7  -- Steven Guiheux <steven.guiheux@somewhere.com>  Mon, 22 Mar 2010 00:37:31 +0100`,
			}
			*(out.(*sdk.VCSContent)) = *b
			return nil, 200, nil
		}).Times(2)

	servicesClients.EXPECT().
		DoJSONRequest(gomock.Any(), "GET", "/vcs/bitbucketserver/repos/my/repo", gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(
			func(ctx context.Context, method, path string, in interface{}, out interface{}, _ interface{}) (http.Header, int, error) {
				b := &sdk.VCSRepo{}
				*(out.(*sdk.VCSRepo)) = *b
				return nil, 200, nil
			},
		).Times(2)

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

	require.Equal(t, "0.9.12-1", wrDB.Contexts.CDS.Version)

	version, err := workflow_v2.LoadWorkflowVersion(ctx, db, wrDB.Contexts.CDS.ProjectKey, wrDB.Contexts.CDS.WorkflowVCSServer, wrDB.Contexts.CDS.WorkflowRepository, wrDB.Contexts.CDS.Workflow, "0.9.12-1")
	require.NoError(t, err)

	require.NotNil(t, version)

	// Update the run and craft it again
	wrDB.Status = sdk.V2WorkflowRunStatusCrafting
	require.NoError(t, workflow_v2.UpdateRun(ctx, db, wrDB))
	require.NoError(t, api.craftWorkflowRunV2(ctx, wrDB.ID))

	wrDB, err = workflow_v2.LoadRunByID(ctx, db, wr.ID)
	require.NoError(t, err)
	versions, err := workflow_v2.LoadAllVerionsByWorkflow(ctx, db, wrDB.Contexts.CDS.ProjectKey, wrDB.Contexts.CDS.WorkflowVCSServer, wrDB.Contexts.CDS.WorkflowRepository, wrDB.Contexts.CDS.Workflow)
	require.NoError(t, err)
	require.Equal(t, 1, len(versions))
	require.Equal(t, "0.9.12-1-1.sha."+wrDB.Contexts.Git.ShaShort, wrDB.Contexts.CDS.Version)
}

func TestCraftWorkflowFromTemplateFail(t *testing.T) {
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

	// Create template
	e := sdk.Entity{
		ProjectKey:          proj.Key,
		Type:                sdk.EntityTypeWorkflowTemplate,
		FilePath:            ".cds/workflow-templates/mytmpl.yml",
		Name:                "myTemplate",
		Commit:              "123456789",
		Ref:                 "refs/heads/master",
		ProjectRepositoryID: repo.ID,
		UserID:              &admin.ID,
		Data: `name: db-schema-model
spec: |-
  stages:
    package: {}
    release:
      needs: [lab]
  jobs:
    build:
      stage: package
    deploy:
      stage: release`,
	}
	require.NoError(t, entity.Insert(ctx, db, &e))

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
		).Times(2)

	wkName := sdk.RandomString(10)
	wr := sdk.V2WorkflowRun{
		DeprecatedUserID: admin.ID,
		ProjectKey:       proj.Key,
		Status:           sdk.V2WorkflowRunStatusCrafting,
		VCSServerID:      vcsProject.ID,
		RepositoryID:     repo.ID,
		RunNumber:        0,
		RunAttempt:       0,
		WorkflowRef:      "refs/heads/master",
		WorkflowSha:      "123456789",
		WorkflowName:     wkName,
		WorkflowData: sdk.V2WorkflowRunData{
			Workflow: sdk.V2Workflow{
				Name: wkName,
				From: fmt.Sprintf("%s/%s/%s/%s", proj.Key, vcsProject.Name, repo.Name, "myTemplate"),
			},
		},
		RunEvent: sdk.V2WorkflowRunEvent{
			HookType:  sdk.WorkflowHookTypeRepository,
			Payload:   nil,
			Ref:       "refs/heads/main",
			Sha:       "123456789",
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
		Commit:              "123456789",
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
		Commit:              "123456789",
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

	require.Equal(t, sdk.V2WorkflowRunStatusFail, wrDB.Status)
	wrInfos, err := workflow_v2.LoadRunInfosByRunID(ctx, db, wr.ID)

	require.NoError(t, err)
	require.Equal(t, 1, len(wrInfos))

	require.Contains(t, wrInfos[0].Message, "stage release: needs not found lab")
}

func TestComputeJobFromTemplate(t *testing.T) {
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

	// Create template
	e := sdk.Entity{
		ProjectKey:          proj.Key,
		Type:                sdk.EntityTypeWorkflowTemplate,
		FilePath:            ".cds/workflow-templates/mytmpl.yml",
		Name:                "myTemplate",
		Commit:              "123456789",
		Ref:                 "refs/heads/master",
		ProjectRepositoryID: repo.ID,
		UserID:              &admin.ID,
		Data: `name: mytemplate
spec: |-
  jobs:
    build:
    test:
    deploy_env1:
      needs: [build,test]
    deploy_env2:
      needs: [build,test]  `,
	}
	require.NoError(t, entity.Insert(ctx, db, &e))

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
		DeprecatedUserID: admin.ID,
		ProjectKey:       proj.Key,
		Status:           sdk.V2WorkflowRunStatusCrafting,
		VCSServerID:      vcsProject.ID,
		RepositoryID:     repo.ID,
		RunNumber:        0,
		RunAttempt:       0,
		WorkflowRef:      "refs/heads/master",
		WorkflowSha:      "123456789",
		WorkflowName:     wkName,
		WorkflowData: sdk.V2WorkflowRunData{
			Workflow: sdk.V2Workflow{
				Name: wkName,
				Jobs: map[string]sdk.V2Job{
					"root": {},
					"two": {
						From:  fmt.Sprintf("%s/%s/%s/%s", proj.Key, vcsProject.Name, repo.Name, "myTemplate"),
						Needs: []string{"root"},
					},
					"three": {
						Needs: []string{"two"},
					},
				},
			},
		},
		RunEvent: sdk.V2WorkflowRunEvent{
			HookType:  sdk.WorkflowHookTypeRepository,
			Payload:   nil,
			Ref:       "refs/heads/main",
			Sha:       "123456789",
			EventName: sdk.WorkflowHookEventNamePush,
		},
	}
	require.NoError(t, workflow_v2.InsertRun(ctx, db, &wr))

	myWMEnt := sdk.Entity{
		ProjectKey:          proj.Key,
		ProjectRepositoryID: repo.ID,
		Type:                sdk.EntityTypeWorkerModel,
		FilePath:            ".cds/worker-models/myworker-model.yml",
		Name:                "myworker-model",
		Ref:                 "refs/heads/master",
		Commit:              "123456789",
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

	require.Equal(t, sdk.V2WorkflowRunStatusBuilding, wrDB.Status)

	infos, err := workflow_v2.LoadRunInfosByRunID(ctx, db, wrDB.ID)
	require.NoError(t, err)
	t.Logf("%+v", infos)

	for k := range wrDB.WorkflowData.Workflow.Jobs {
		t.Logf("%s", k)
	}
	require.Equal(t, 6, len(wrDB.WorkflowData.Workflow.Jobs))

	buildJob := wrDB.WorkflowData.Workflow.Jobs["build"]
	require.Equal(t, 1, len(buildJob.Needs))
	require.Equal(t, "root", buildJob.Needs[0])
	testJob := wrDB.WorkflowData.Workflow.Jobs["test"]
	require.Equal(t, 1, len(testJob.Needs))
	require.Equal(t, "root", testJob.Needs[0])

	threeJob := wrDB.WorkflowData.Workflow.Jobs["three"]
	require.Equal(t, 2, len(threeJob.Needs))
	require.True(t, slices.Contains(threeJob.Needs, "deploy_env1"))
	require.True(t, slices.Contains(threeJob.Needs, "deploy_env2"))

}

func TestComputeJobFromMultiTemplate(t *testing.T) {
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

	// Create template
	e := sdk.Entity{
		ProjectKey:          proj.Key,
		Type:                sdk.EntityTypeWorkflowTemplate,
		FilePath:            ".cds/workflow-templates/mytmpl.yml",
		Name:                "myTemplate",
		Commit:              "123456789",
		Ref:                 "refs/heads/master",
		ProjectRepositoryID: repo.ID,
		UserID:              &admin.ID,
		Data: `name: mytemplate
spec: |-
  jobs:
    build:
    test:
    deploy_env1:
      needs: [build,test]
    deploy_env2:
      needs: [build,test]  `,
	}
	require.NoError(t, entity.Insert(ctx, db, &e))

	eTmpl2 := sdk.Entity{
		ProjectKey:          proj.Key,
		Type:                sdk.EntityTypeWorkflowTemplate,
		FilePath:            ".cds/workflow-templates/mytmpl2.yml",
		Name:                "myJobTemplate",
		Commit:              "123456789",
		Ref:                 "refs/heads/master",
		ProjectRepositoryID: repo.ID,
		UserID:              &admin.ID,
		Data: `name: mytemplate
spec: |-
  jobs:
    smoke_1:
    smoke_2:
    smoke_3:
      needs: [smoke_1]
    smoke_4:
      needs: [smoke_2]  `,
	}
	require.NoError(t, entity.Insert(ctx, db, &eTmpl2))

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
		DeprecatedUserID: admin.ID,
		ProjectKey:       proj.Key,
		Status:           sdk.V2WorkflowRunStatusCrafting,
		VCSServerID:      vcsProject.ID,
		RepositoryID:     repo.ID,
		RunNumber:        0,
		RunAttempt:       0,
		WorkflowRef:      "refs/heads/master",
		WorkflowSha:      "123456789",
		WorkflowName:     wkName,
		WorkflowData: sdk.V2WorkflowRunData{
			Workflow: sdk.V2Workflow{
				Name: wkName,
				Jobs: map[string]sdk.V2Job{
					"root": {},
					"two": {
						From:  fmt.Sprintf("%s/%s/%s/%s", proj.Key, vcsProject.Name, repo.Name, "myTemplate"),
						Needs: []string{"root"},
					},
					"three": {
						From:  fmt.Sprintf("%s/%s/%s/%s", proj.Key, vcsProject.Name, repo.Name, "myJobTemplate"),
						Needs: []string{"two"},
					},
					"four": {
						Needs: []string{"three"},
					},
				},
			},
		},
		RunEvent: sdk.V2WorkflowRunEvent{
			HookType:  sdk.WorkflowHookTypeRepository,
			Payload:   nil,
			Ref:       "refs/heads/main",
			Sha:       "123456789",
			EventName: sdk.WorkflowHookEventNamePush,
		},
	}
	require.NoError(t, workflow_v2.InsertRun(ctx, db, &wr))

	myWMEnt := sdk.Entity{
		ProjectKey:          proj.Key,
		ProjectRepositoryID: repo.ID,
		Type:                sdk.EntityTypeWorkerModel,
		FilePath:            ".cds/worker-models/myworker-model.yml",
		Name:                "myworker-model",
		Ref:                 "refs/heads/master",
		Commit:              "123456789",
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

	require.Equal(t, sdk.V2WorkflowRunStatusBuilding, wrDB.Status)

	infos, err := workflow_v2.LoadRunInfosByRunID(ctx, db, wrDB.ID)
	require.NoError(t, err)
	t.Logf("%+v", infos)

	for k := range wrDB.WorkflowData.Workflow.Jobs {
		t.Logf("%s", k)
	}
	require.Equal(t, 10, len(wrDB.WorkflowData.Workflow.Jobs))

	buildJob := wrDB.WorkflowData.Workflow.Jobs["build"]
	require.Equal(t, 1, len(buildJob.Needs))
	require.Equal(t, "root", buildJob.Needs[0])
	testJob := wrDB.WorkflowData.Workflow.Jobs["test"]
	require.Equal(t, 1, len(testJob.Needs))
	require.Equal(t, "root", testJob.Needs[0])

	smoke1 := wrDB.WorkflowData.Workflow.Jobs["smoke_1"]
	require.True(t, slices.Contains(smoke1.Needs, "deploy_env1"))
	require.True(t, slices.Contains(smoke1.Needs, "deploy_env2"))

	smoke2 := wrDB.WorkflowData.Workflow.Jobs["smoke_2"]
	require.True(t, slices.Contains(smoke2.Needs, "deploy_env1"))
	require.True(t, slices.Contains(smoke2.Needs, "deploy_env2"))

	fourJob := wrDB.WorkflowData.Workflow.Jobs["four"]
	require.Equal(t, 2, len(fourJob.Needs))
	require.True(t, slices.Contains(fourJob.Needs, "smoke_3"))
	require.True(t, slices.Contains(fourJob.Needs, "smoke_4"))

}

func TestComputeJobFromTemplate_DuplicateJob(t *testing.T) {
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

	// Create template
	e := sdk.Entity{
		ProjectKey:          proj.Key,
		Type:                sdk.EntityTypeWorkflowTemplate,
		FilePath:            ".cds/workflow-templates/mytmpl.yml",
		Name:                "myTemplate",
		Commit:              "123456789",
		Ref:                 "refs/heads/master",
		ProjectRepositoryID: repo.ID,
		UserID:              &admin.ID,
		Data: `name: mytemplate
spec: |-
  jobs:
    build:
    test:
    deploy_env1:
      needs: [build,test]
    deploy_env2:
      needs: [build,test]  `,
	}
	require.NoError(t, entity.Insert(ctx, db, &e))

	eTmpl2 := sdk.Entity{
		ProjectKey:          proj.Key,
		Type:                sdk.EntityTypeWorkflowTemplate,
		FilePath:            ".cds/workflow-templates/mytmpl2.yml",
		Name:                "myJobTemplate",
		Commit:              "123456789",
		Ref:                 "refs/heads/master",
		ProjectRepositoryID: repo.ID,
		UserID:              &admin.ID,
		Data: `name: mytemplate
spec: |-
  jobs:
    smoke_1:
    smoke_2:
    smoke_3:
      needs: [smoke_1]
    smoke_4:
      needs: [smoke_2]  `,
	}
	require.NoError(t, entity.Insert(ctx, db, &eTmpl2))

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
		DeprecatedUserID: admin.ID,
		ProjectKey:       proj.Key,
		Status:           sdk.V2WorkflowRunStatusCrafting,
		VCSServerID:      vcsProject.ID,
		RepositoryID:     repo.ID,
		RunNumber:        0,
		RunAttempt:       0,
		WorkflowRef:      "refs/heads/master",
		WorkflowSha:      "123456789",
		WorkflowName:     wkName,
		WorkflowData: sdk.V2WorkflowRunData{
			Workflow: sdk.V2Workflow{
				Name: wkName,
				Jobs: map[string]sdk.V2Job{
					"root": {},
					"two": {
						From:  fmt.Sprintf("%s/%s/%s/%s", proj.Key, vcsProject.Name, repo.Name, "myTemplate"),
						Needs: []string{"root"},
					},
					"three": {
						From:  fmt.Sprintf("%s/%s/%s/%s", proj.Key, vcsProject.Name, repo.Name, "myTemplate"),
						Needs: []string{"two"},
					},
					"four": {
						Needs: []string{"three"},
					},
				},
			},
		},
		RunEvent: sdk.V2WorkflowRunEvent{
			HookType:  sdk.WorkflowHookTypeRepository,
			Payload:   nil,
			Ref:       "refs/heads/main",
			Sha:       "123456789",
			EventName: sdk.WorkflowHookEventNamePush,
		},
	}
	require.NoError(t, workflow_v2.InsertRun(ctx, db, &wr))

	myWMEnt := sdk.Entity{
		ProjectKey:          proj.Key,
		ProjectRepositoryID: repo.ID,
		Type:                sdk.EntityTypeWorkerModel,
		FilePath:            ".cds/worker-models/myworker-model.yml",
		Name:                "myworker-model",
		Ref:                 "refs/heads/master",
		Commit:              "123456789",
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

	require.Equal(t, sdk.V2WorkflowRunStatusFail, wrDB.Status)

	infos, err := workflow_v2.LoadRunInfosByRunID(ctx, db, wrDB.ID)
	require.NoError(t, err)

	require.Equal(t, 1, len(infos))
	require.Contains(t, infos[0].Message, "already exist in the parent workflow")
	t.Logf(infos[0].Message)
}

func TestComputeJobFromTemplate_AddingStageOnNonStagedWorkflow(t *testing.T) {
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

	// Create template
	e := sdk.Entity{
		ProjectKey:          proj.Key,
		Type:                sdk.EntityTypeWorkflowTemplate,
		FilePath:            ".cds/workflow-templates/mytmpl.yml",
		Name:                "myTemplate",
		Commit:              "123456789",
		Ref:                 "refs/heads/master",
		ProjectRepositoryID: repo.ID,
		UserID:              &admin.ID,
		Data: `name: mytemplate
spec: |-
  stages:
    toto: {}
  jobs:
    build:
      stage: toto
    test:
      stage: toto
    deploy_env1:
      stage: toto
      needs: [build,test]
    deploy_env2:
      stage: toto
      needs: [build,test]  `,
	}
	require.NoError(t, entity.Insert(ctx, db, &e))

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
		DeprecatedUserID: admin.ID,
		ProjectKey:       proj.Key,
		Status:           sdk.V2WorkflowRunStatusCrafting,
		VCSServerID:      vcsProject.ID,
		RepositoryID:     repo.ID,
		RunNumber:        0,
		RunAttempt:       0,
		WorkflowRef:      "refs/heads/master",
		WorkflowSha:      "123456789",
		WorkflowName:     wkName,
		WorkflowData: sdk.V2WorkflowRunData{
			Workflow: sdk.V2Workflow{
				Name: wkName,
				Jobs: map[string]sdk.V2Job{
					"root": {},
					"two": {
						From:  fmt.Sprintf("%s/%s/%s/%s", proj.Key, vcsProject.Name, repo.Name, "myTemplate"),
						Needs: []string{"root"},
					},
					"three": {
						Needs: []string{"two"},
					},
				},
			},
		},
		RunEvent: sdk.V2WorkflowRunEvent{
			HookType:  sdk.WorkflowHookTypeRepository,
			Payload:   nil,
			Ref:       "refs/heads/main",
			Sha:       "123456789",
			EventName: sdk.WorkflowHookEventNamePush,
		},
	}
	require.NoError(t, workflow_v2.InsertRun(ctx, db, &wr))

	myWMEnt := sdk.Entity{
		ProjectKey:          proj.Key,
		ProjectRepositoryID: repo.ID,
		Type:                sdk.EntityTypeWorkerModel,
		FilePath:            ".cds/worker-models/myworker-model.yml",
		Name:                "myworker-model",
		Ref:                 "refs/heads/master",
		Commit:              "123456789",
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

	require.Equal(t, sdk.V2WorkflowRunStatusFail, wrDB.Status)

	infos, err := workflow_v2.LoadRunInfosByRunID(ctx, db, wrDB.ID)
	require.NoError(t, err)
	require.Len(t, infos, 2)
	for _, i := range infos {
		require.Contains(t, i.Message, "missing stage on job")
	}
}

func TestComputeConcurrency(t *testing.T) {
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
		DeprecatedUserID: admin.ID,
		ProjectKey:       proj.Key,
		Status:           sdk.V2WorkflowRunStatusCrafting,
		VCSServerID:      vcsProject.ID,
		RepositoryID:     repo.ID,
		RunNumber:        0,
		RunAttempt:       0,
		WorkflowRef:      "refs/heads/master",
		WorkflowSha:      "123456789",
		WorkflowName:     wkName,
		WorkflowData: sdk.V2WorkflowRunData{
			Workflow: sdk.V2Workflow{
				Name: wkName,
				Jobs: map[string]sdk.V2Job{
					"root": {
						Concurrency: "${{git.ref_name}}",
						RunsOn: sdk.V2JobRunsOn{
							Model: "${{ blabla }}",
						},
						Region: "build",
						Steps: []sdk.ActionStep{
							{
								Run: "echo toto",
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
			Sha:       "123456789",
			EventName: sdk.WorkflowHookEventNamePush,
		},
	}
	require.NoError(t, workflow_v2.InsertRun(ctx, db, &wr))

	myWMEnt := sdk.Entity{
		ProjectKey:          proj.Key,
		ProjectRepositoryID: repo.ID,
		Type:                sdk.EntityTypeWorkerModel,
		FilePath:            ".cds/worker-models/myworker-model.yml",
		Name:                "myworker-model",
		Ref:                 "refs/heads/master",
		Commit:              "123456789",
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

	infos, err := workflow_v2.LoadRunInfosByRunID(ctx, db, wrDB.ID)
	require.NoError(t, err)
	t.Logf("%+v", infos)

	require.Equal(t, "main", wrDB.WorkflowData.Workflow.Jobs["root"].Concurrency)
	require.Equal(t, sdk.V2WorkflowRunStatusBuilding, wrDB.Status)
}
