package api

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/rbac"
	"github.com/ovh/cds/engine/api/region"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/engine/api/workflow_v2"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
)

func TestRetrieveRunJobToUnlocked_WorkflowScoped_OldestFirst(t *testing.T) {
	api, db, _ := newTestAPI(t)

	admin, _ := assets.InsertAdminUser(t, db)
	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))
	vcsServer := assets.InsertTestVCSProject(t, db, proj.ID, "github", "github")
	repo := assets.InsertTestProjectRepository(t, db, proj.Key, vcsServer.ID, sdk.RandomString(10))

	wr := sdk.V2WorkflowRun{
		ID:           sdk.UUID(),
		ProjectKey:   proj.Key,
		VCSServerID:  vcsServer.ID,
		VCSServer:    vcsServer.Name,
		RepositoryID: repo.ID,
		Repository:   repo.Name,
		WorkflowName: "myworkflow",
		WorkflowSha:  "azerty",
		WorkflowRef:  "refs/heads/main",
		Status:       sdk.V2WorkflowRunStatusBuilding,
		RunNumber:    1,
		RunAttempt:   1,
		Started:      time.Now(),
		LastModified: time.Now(),
		WorkflowData: sdk.V2WorkflowRunData{},
		Contexts:     sdk.WorkflowRunContext{},
		Initiator:    &sdk.V2Initiator{UserID: admin.ID},
	}
	require.NoError(t, workflow_v2.InsertRun(context.TODO(), db, &wr))

	wrOld := sdk.V2WorkflowRun{
		ID:           sdk.UUID(),
		ProjectKey:   proj.Key,
		VCSServerID:  vcsServer.ID,
		VCSServer:    vcsServer.Name,
		RepositoryID: repo.ID,
		Repository:   repo.Name,
		WorkflowName: "myworkflow",
		WorkflowSha:  "azerty",
		WorkflowRef:  "refs/heads/main",
		Status:       sdk.V2WorkflowRunStatusBlocked,
		RunNumber:    2,
		RunAttempt:   1,
		Started:      time.Now(),
		LastModified: time.Now(),
		WorkflowData: sdk.V2WorkflowRunData{},
		Contexts:     sdk.WorkflowRunContext{},
		Initiator:    &sdk.V2Initiator{UserID: admin.ID},
		Concurrency: &sdk.V2RunConcurrency{
			WorkflowConcurrency: sdk.WorkflowConcurrency{
				Name:             "main",
				Order:            sdk.ConcurrencyOrderOldestFirst,
				Pool:             3,
				CancelInProgress: false,
			},
			Scope: sdk.V2RunConcurrencyScopeWorkflow,
		},
	}
	require.NoError(t, workflow_v2.InsertRun(context.TODO(), db, &wrOld))

	jobRunOld := sdk.V2WorkflowRunJob{
		ID:            sdk.UUID(),
		JobID:         sdk.RandomString(10),
		WorkflowRunID: wr.ID,
		ProjectKey:    wr.ProjectKey,
		VCSServer:     wr.VCSServer,
		Repository:    wr.Repository,
		WorkflowName:  wr.WorkflowName,
		RunNumber:     wr.RunNumber,
		RunAttempt:    wr.RunAttempt,
		Status:        sdk.StatusBlocked,
		Queued:        time.Now().Add(1 * time.Second),
		Job:           sdk.V2Job{},
		Concurrency: &sdk.V2RunConcurrency{
			WorkflowConcurrency: sdk.WorkflowConcurrency{
				Name:             "main",
				Order:            sdk.ConcurrencyOrderOldestFirst,
				Pool:             3,
				CancelInProgress: false,
			},
			Scope: sdk.V2RunConcurrencyScopeWorkflow,
		},
		Initiator: *wr.Initiator,
	}
	jobRunOld2 := sdk.V2WorkflowRunJob{
		ID:            sdk.UUID(),
		JobID:         sdk.RandomString(10),
		WorkflowRunID: wr.ID,
		ProjectKey:    wr.ProjectKey,
		VCSServer:     wr.VCSServer,
		Repository:    wr.Repository,
		WorkflowName:  wr.WorkflowName,
		RunNumber:     wr.RunNumber,
		RunAttempt:    wr.RunAttempt,
		Status:        sdk.StatusBlocked,
		Queued:        time.Now().Add(10 * time.Second),
		Job:           sdk.V2Job{},
		Concurrency: &sdk.V2RunConcurrency{
			WorkflowConcurrency: sdk.WorkflowConcurrency{
				Name:             "main",
				Order:            sdk.ConcurrencyOrderOldestFirst,
				Pool:             3,
				CancelInProgress: false,
			},
			Scope: sdk.V2RunConcurrencyScopeWorkflow,
		},
		Initiator: *wr.Initiator,
	}
	jobRunNew := sdk.V2WorkflowRunJob{
		ID:            sdk.UUID(),
		JobID:         sdk.RandomString(10),
		WorkflowRunID: wr.ID,
		ProjectKey:    wr.ProjectKey,
		VCSServer:     wr.VCSServer,
		Repository:    wr.Repository,
		WorkflowName:  wr.WorkflowName,
		RunNumber:     wr.RunNumber,
		RunAttempt:    wr.RunAttempt,
		Status:        sdk.StatusBlocked,
		Queued:        time.Now().Add(1 * time.Minute),
		Job:           sdk.V2Job{},
		Concurrency: &sdk.V2RunConcurrency{
			WorkflowConcurrency: sdk.WorkflowConcurrency{
				Name:             "main",
				Order:            sdk.ConcurrencyOrderOldestFirst,
				Pool:             3,
				CancelInProgress: false,
			},
			Scope: sdk.V2RunConcurrencyScopeWorkflow,
		},
		Initiator: *wr.Initiator,
	}

	require.NoError(t, workflow_v2.InsertRunJob(context.TODO(), db, &jobRunNew))
	require.NoError(t, workflow_v2.InsertRunJob(context.TODO(), db, &jobRunOld))
	require.NoError(t, workflow_v2.InsertRunJob(context.TODO(), db, &jobRunOld2))

	objs, _, err := retrieveRunObjectsToUnLocked(context.TODO(), db.DbMap, jobRunNew.ProjectKey, jobRunNew.VCSServer, jobRunNew.Repository, jobRunNew.WorkflowName, *jobRunNew.Concurrency)
	require.NoError(t, err)
	t.Logf(">>>%+v", objs)
	require.Equal(t, 3, len(objs))
	require.Equal(t, wrOld.ID, objs[0].ID)
	require.Equal(t, workflow_v2.ConcurrencyObjectTypeWorkflow, objs[0].Type)
	require.Equal(t, jobRunOld.ID, objs[1].ID)
	require.Equal(t, workflow_v2.ConcurrencyObjectTypeJob, objs[1].Type)
	require.Equal(t, jobRunOld2.ID, objs[2].ID)
	require.Equal(t, workflow_v2.ConcurrencyObjectTypeJob, objs[2].Type)
}

func TestRetrieveRunJobToUnlocked_WorkflowScoped_NewestFirst(t *testing.T) {
	api, db, _ := newTestAPI(t)

	admin, _ := assets.InsertAdminUser(t, db)
	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))
	vcsServer := assets.InsertTestVCSProject(t, db, proj.ID, "github", "github")
	repo := assets.InsertTestProjectRepository(t, db, proj.Key, vcsServer.ID, sdk.RandomString(10))

	wr := sdk.V2WorkflowRun{
		ID:           sdk.UUID(),
		ProjectKey:   proj.Key,
		VCSServerID:  vcsServer.ID,
		VCSServer:    vcsServer.Name,
		RepositoryID: repo.ID,
		Repository:   repo.Name,
		WorkflowName: "myworkflow",
		WorkflowSha:  "azerty",
		WorkflowRef:  "refs/heads/main",
		Status:       sdk.V2WorkflowRunStatusBuilding,
		RunNumber:    1,
		RunAttempt:   1,
		Started:      time.Now(),
		LastModified: time.Now(),
		WorkflowData: sdk.V2WorkflowRunData{},
		Contexts:     sdk.WorkflowRunContext{},
		Initiator:    &sdk.V2Initiator{UserID: admin.ID},
	}
	require.NoError(t, workflow_v2.InsertRun(context.TODO(), db, &wr))

	wrNew := sdk.V2WorkflowRun{
		ID:           sdk.UUID(),
		ProjectKey:   proj.Key,
		VCSServerID:  vcsServer.ID,
		VCSServer:    vcsServer.Name,
		RepositoryID: repo.ID,
		Repository:   repo.Name,
		WorkflowName: "myworkflow",
		WorkflowSha:  "azerty",
		WorkflowRef:  "refs/heads/main",
		Status:       sdk.V2WorkflowRunStatusBlocked,
		RunNumber:    2,
		RunAttempt:   1,
		Started:      time.Now(),
		LastModified: time.Now(),
		WorkflowData: sdk.V2WorkflowRunData{},
		Contexts:     sdk.WorkflowRunContext{},
		Initiator:    &sdk.V2Initiator{UserID: admin.ID},
		Concurrency: &sdk.V2RunConcurrency{
			WorkflowConcurrency: sdk.WorkflowConcurrency{
				Name:             "main",
				Order:            sdk.ConcurrencyOrderNewestFirst,
				Pool:             3,
				CancelInProgress: false,
			},
			Scope: sdk.V2RunConcurrencyScopeWorkflow,
		},
	}
	require.NoError(t, workflow_v2.InsertRun(context.TODO(), db, &wrNew))
	_, err := db.Exec("UPDATE v2_workflow_run SET last_modified = $1", time.Now().Add(10*time.Minute))
	require.NoError(t, err)

	jobRunOld := sdk.V2WorkflowRunJob{
		ID:            sdk.UUID(),
		JobID:         sdk.RandomString(10),
		WorkflowRunID: wr.ID,
		ProjectKey:    wr.ProjectKey,
		VCSServer:     wr.VCSServer,
		Repository:    wr.Repository,
		WorkflowName:  wr.WorkflowName,
		RunNumber:     wr.RunNumber,
		RunAttempt:    wr.RunAttempt,
		Status:        sdk.StatusBlocked,
		Queued:        time.Now(),
		Job:           sdk.V2Job{},
		Concurrency: &sdk.V2RunConcurrency{
			WorkflowConcurrency: sdk.WorkflowConcurrency{
				Name:             "main",
				Order:            sdk.ConcurrencyOrderNewestFirst,
				Pool:             3,
				CancelInProgress: false,
			},
			Scope: sdk.V2RunConcurrencyScopeWorkflow,
		},
		Initiator: *wr.Initiator,
	}
	jobRunOld2 := sdk.V2WorkflowRunJob{
		ID:            sdk.UUID(),
		JobID:         sdk.RandomString(10),
		WorkflowRunID: wr.ID,
		ProjectKey:    wr.ProjectKey,
		VCSServer:     wr.VCSServer,
		Repository:    wr.Repository,
		WorkflowName:  wr.WorkflowName,
		RunNumber:     wr.RunNumber,
		RunAttempt:    wr.RunAttempt,
		Status:        sdk.StatusBlocked,
		Queued:        time.Now().Add(30 * time.Second),
		Job:           sdk.V2Job{},
		Concurrency: &sdk.V2RunConcurrency{
			WorkflowConcurrency: sdk.WorkflowConcurrency{
				Name:             "main",
				Order:            sdk.ConcurrencyOrderNewestFirst,
				Pool:             3,
				CancelInProgress: false,
			},
			Scope: sdk.V2RunConcurrencyScopeWorkflow,
		},
		Initiator: *wr.Initiator,
	}
	jobRunNew := sdk.V2WorkflowRunJob{
		ID:            sdk.UUID(),
		JobID:         sdk.RandomString(10),
		WorkflowRunID: wr.ID,
		ProjectKey:    wr.ProjectKey,
		VCSServer:     wr.VCSServer,
		Repository:    wr.Repository,
		WorkflowName:  wr.WorkflowName,
		RunNumber:     wr.RunNumber,
		RunAttempt:    wr.RunAttempt,
		Status:        sdk.StatusBlocked,
		Queued:        time.Now().Add(1 * time.Minute),
		Job:           sdk.V2Job{},
		Concurrency: &sdk.V2RunConcurrency{
			WorkflowConcurrency: sdk.WorkflowConcurrency{
				Name:             "main",
				Order:            sdk.ConcurrencyOrderNewestFirst,
				Pool:             3,
				CancelInProgress: false,
			},
			Scope: sdk.V2RunConcurrencyScopeWorkflow,
		},
		Initiator: *wr.Initiator,
	}

	require.NoError(t, workflow_v2.InsertRunJob(context.TODO(), db, &jobRunNew))
	require.NoError(t, workflow_v2.InsertRunJob(context.TODO(), db, &jobRunOld))
	require.NoError(t, workflow_v2.InsertRunJob(context.TODO(), db, &jobRunOld2))

	objs, _, err := retrieveRunObjectsToUnLocked(context.TODO(), db.DbMap, jobRunNew.ProjectKey, jobRunNew.VCSServer, jobRunNew.Repository, jobRunNew.WorkflowName, *jobRunNew.Concurrency)
	require.NoError(t, err)
	require.Equal(t, 3, len(objs))
	require.Equal(t, wrNew.ID, objs[0].ID)
	require.Equal(t, workflow_v2.ConcurrencyObjectTypeWorkflow, objs[0].Type)
	require.Equal(t, jobRunNew.ID, objs[1].ID)
	require.Equal(t, workflow_v2.ConcurrencyObjectTypeJob, objs[1].Type)
	require.Equal(t, jobRunOld2.ID, objs[2].ID)
	require.Equal(t, workflow_v2.ConcurrencyObjectTypeJob, objs[2].Type)
}

func TestCheckJobWorkflowConcurrency_DefaultRules(t *testing.T) {
	api, db, _ := newTestAPI(t)

	admin, _ := assets.InsertAdminUser(t, db)
	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))
	vcsServer := assets.InsertTestVCSProject(t, db, proj.ID, "github", "github")
	repo := assets.InsertTestProjectRepository(t, db, proj.Key, vcsServer.ID, sdk.RandomString(10))

	wr := sdk.V2WorkflowRun{
		ID:           sdk.UUID(),
		ProjectKey:   proj.Key,
		VCSServerID:  vcsServer.ID,
		VCSServer:    vcsServer.Name,
		RepositoryID: repo.ID,
		Repository:   repo.Name,
		WorkflowName: "myworkflow",
		WorkflowSha:  "azerty",
		WorkflowRef:  "refs/heads/main",
		Status:       sdk.V2WorkflowRunStatusBuilding,
		RunNumber:    1,
		RunAttempt:   1,
		Started:      time.Now(),
		LastModified: time.Now(),
		WorkflowData: sdk.V2WorkflowRunData{},
		Contexts:     sdk.WorkflowRunContext{},
		Initiator:    &sdk.V2Initiator{UserID: admin.ID},
	}
	require.NoError(t, workflow_v2.InsertRun(context.TODO(), db, &wr))

	jobRunOld := sdk.V2WorkflowRunJob{
		ID:            sdk.UUID(),
		JobID:         sdk.RandomString(10),
		WorkflowRunID: wr.ID,
		ProjectKey:    wr.ProjectKey,
		VCSServer:     wr.VCSServer,
		Repository:    wr.Repository,
		WorkflowName:  wr.WorkflowName,
		RunNumber:     wr.RunNumber,
		RunAttempt:    wr.RunAttempt,
		Status:        sdk.StatusBlocked,
		Queued:        time.Now(),
		Job:           sdk.V2Job{},
		Concurrency: &sdk.V2RunConcurrency{
			WorkflowConcurrency: sdk.WorkflowConcurrency{
				Name:             "main",
				Order:            sdk.ConcurrencyOrderOldestFirst,
				Pool:             3,
				CancelInProgress: true,
			},
			Scope: sdk.V2RunConcurrencyScopeWorkflow,
		},
		Initiator: *wr.Initiator,
	}
	jobRunNew := sdk.V2WorkflowRunJob{
		ID:            sdk.UUID(),
		JobID:         sdk.RandomString(10),
		WorkflowRunID: wr.ID,
		ProjectKey:    wr.ProjectKey,
		VCSServer:     wr.VCSServer,
		Repository:    wr.Repository,
		WorkflowName:  wr.WorkflowName,
		RunNumber:     wr.RunNumber,
		RunAttempt:    wr.RunAttempt,
		Status:        sdk.StatusBlocked,
		Queued:        time.Now().Add(1 * time.Minute),
		Job:           sdk.V2Job{},
		Concurrency: &sdk.V2RunConcurrency{
			WorkflowConcurrency: sdk.WorkflowConcurrency{
				Name:             "main",
				Order:            sdk.ConcurrencyOrderNewestFirst,
				Pool:             10,
				CancelInProgress: false,
			},
			Scope: sdk.V2RunConcurrencyScopeWorkflow,
		},
		Initiator: *wr.Initiator,
	}
	require.NoError(t, workflow_v2.InsertRunJob(context.TODO(), db, &jobRunNew))
	require.NoError(t, workflow_v2.InsertRunJob(context.TODO(), db, &jobRunOld))

	rule, nbBuilding, nbBlocking, err := checkWorkflowScopedConcurrency(context.TODO(), db, wr.ProjectKey, wr.VCSServer, wr.Repository, wr.WorkflowName, jobRunNew.Concurrency.WorkflowConcurrency)
	require.NoError(t, err)

	require.Equal(t, int64(0), nbBuilding)
	require.Equal(t, int64(2), nbBlocking)

	require.Equal(t, sdk.ConcurrencyOrderOldestFirst, rule.Order)
	require.Equal(t, int64(3), rule.Pool)
	require.False(t, rule.CancelInProgress)
}

func TestRetrieveRunJobToUnlocked_ProjectScoped_OldestFirst(t *testing.T) {
	api, db, _ := newTestAPI(t)

	admin, _ := assets.InsertAdminUser(t, db)
	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))
	vcsServer := assets.InsertTestVCSProject(t, db, proj.ID, "github", "github")
	repo := assets.InsertTestProjectRepository(t, db, proj.Key, vcsServer.ID, sdk.RandomString(10))

	wr := sdk.V2WorkflowRun{
		ID:           sdk.UUID(),
		ProjectKey:   proj.Key,
		VCSServerID:  vcsServer.ID,
		VCSServer:    vcsServer.Name,
		RepositoryID: repo.ID,
		Repository:   repo.Name,
		WorkflowName: "myworkflow",
		WorkflowSha:  "azerty",
		WorkflowRef:  "refs/heads/main",
		Status:       sdk.V2WorkflowRunStatusBuilding,
		RunNumber:    1,
		RunAttempt:   1,
		Started:      time.Now(),
		LastModified: time.Now(),
		WorkflowData: sdk.V2WorkflowRunData{},
		Contexts:     sdk.WorkflowRunContext{},
		Initiator:    &sdk.V2Initiator{UserID: admin.ID},
	}
	require.NoError(t, workflow_v2.InsertRun(context.TODO(), db, &wr))

	wrOld := sdk.V2WorkflowRun{
		ID:           sdk.UUID(),
		ProjectKey:   proj.Key,
		VCSServerID:  vcsServer.ID,
		VCSServer:    vcsServer.Name,
		RepositoryID: repo.ID,
		Repository:   repo.Name,
		WorkflowName: "myworkflow",
		WorkflowSha:  "azerty",
		WorkflowRef:  "refs/heads/main",
		Status:       sdk.V2WorkflowRunStatusBlocked,
		RunNumber:    2,
		RunAttempt:   1,
		Started:      time.Now(),
		LastModified: time.Now(),
		WorkflowData: sdk.V2WorkflowRunData{},
		Contexts:     sdk.WorkflowRunContext{},
		Initiator:    &sdk.V2Initiator{UserID: admin.ID},
		Concurrency: &sdk.V2RunConcurrency{
			WorkflowConcurrency: sdk.WorkflowConcurrency{
				Name:             "main",
				Order:            sdk.ConcurrencyOrderOldestFirst,
				Pool:             3,
				CancelInProgress: false,
			},
			Scope: sdk.V2RunConcurrencyScopeProject,
		},
	}
	require.NoError(t, workflow_v2.InsertRun(context.TODO(), db, &wrOld))

	jobRunOld := sdk.V2WorkflowRunJob{
		ID:            sdk.UUID(),
		JobID:         sdk.RandomString(10),
		WorkflowRunID: wr.ID,
		ProjectKey:    wr.ProjectKey,
		VCSServer:     wr.VCSServer,
		Repository:    wr.Repository,
		WorkflowName:  wr.WorkflowName,
		RunNumber:     wr.RunNumber,
		RunAttempt:    wr.RunAttempt,
		Status:        sdk.StatusBlocked,
		Queued:        time.Now().Add(1 * time.Second),
		Job:           sdk.V2Job{},
		Concurrency: &sdk.V2RunConcurrency{
			WorkflowConcurrency: sdk.WorkflowConcurrency{
				Name:             "main",
				Order:            sdk.ConcurrencyOrderOldestFirst,
				Pool:             3,
				CancelInProgress: false,
			},
			Scope: sdk.V2RunConcurrencyScopeProject,
		},
		Initiator: *wr.Initiator,
	}
	jobRunOld2 := sdk.V2WorkflowRunJob{
		ID:            sdk.UUID(),
		JobID:         sdk.RandomString(10),
		WorkflowRunID: wr.ID,
		ProjectKey:    wr.ProjectKey,
		VCSServer:     wr.VCSServer,
		Repository:    wr.Repository,
		WorkflowName:  wr.WorkflowName,
		RunNumber:     wr.RunNumber,
		RunAttempt:    wr.RunAttempt,
		Status:        sdk.StatusBlocked,
		Queued:        time.Now().Add(30 * time.Second),
		Job:           sdk.V2Job{},
		Concurrency: &sdk.V2RunConcurrency{
			WorkflowConcurrency: sdk.WorkflowConcurrency{
				Name:             "main",
				Order:            sdk.ConcurrencyOrderOldestFirst,
				Pool:             3,
				CancelInProgress: false,
			},
			Scope: sdk.V2RunConcurrencyScopeProject,
		},
		Initiator: *wr.Initiator,
	}
	jobRunNew := sdk.V2WorkflowRunJob{
		ID:            sdk.UUID(),
		JobID:         sdk.RandomString(10),
		WorkflowRunID: wr.ID,
		ProjectKey:    wr.ProjectKey,
		VCSServer:     wr.VCSServer,
		Repository:    wr.Repository,
		WorkflowName:  wr.WorkflowName,
		RunNumber:     wr.RunNumber,
		RunAttempt:    wr.RunAttempt,
		Status:        sdk.StatusBlocked,
		Queued:        time.Now().Add(1 * time.Minute),
		Job:           sdk.V2Job{},
		Concurrency: &sdk.V2RunConcurrency{
			WorkflowConcurrency: sdk.WorkflowConcurrency{
				Name:             "main",
				Order:            sdk.ConcurrencyOrderOldestFirst,
				Pool:             3,
				CancelInProgress: false,
			},
			Scope: sdk.V2RunConcurrencyScopeProject,
		},
		Initiator: *wr.Initiator,
	}

	require.NoError(t, workflow_v2.InsertRunJob(context.TODO(), db, &jobRunNew))
	require.NoError(t, workflow_v2.InsertRunJob(context.TODO(), db, &jobRunOld))
	require.NoError(t, workflow_v2.InsertRunJob(context.TODO(), db, &jobRunOld2))

	objs, _, err := retrieveRunObjectsToUnLocked(context.TODO(), db.DbMap, jobRunNew.ProjectKey, jobRunNew.VCSServer, jobRunNew.Repository, jobRunNew.WorkflowName, *jobRunNew.Concurrency)
	require.NoError(t, err)
	require.Equal(t, 3, len(objs))
	require.Equal(t, wrOld.ID, objs[0].ID)
	require.Equal(t, workflow_v2.ConcurrencyObjectTypeWorkflow, objs[0].Type)
	require.Equal(t, jobRunOld.ID, objs[1].ID)
	require.Equal(t, workflow_v2.ConcurrencyObjectTypeJob, objs[1].Type)
	require.Equal(t, jobRunOld2.ID, objs[2].ID)
	require.Equal(t, workflow_v2.ConcurrencyObjectTypeJob, objs[2].Type)
}

func TestRetrieveRunJobToUnlocked_ProjectScoped_NewestFirst(t *testing.T) {
	api, db, _ := newTestAPI(t)

	admin, _ := assets.InsertAdminUser(t, db)
	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))
	vcsServer := assets.InsertTestVCSProject(t, db, proj.ID, "github", "github")
	repo := assets.InsertTestProjectRepository(t, db, proj.Key, vcsServer.ID, sdk.RandomString(10))

	wr := sdk.V2WorkflowRun{
		ID:           sdk.UUID(),
		ProjectKey:   proj.Key,
		VCSServerID:  vcsServer.ID,
		VCSServer:    vcsServer.Name,
		RepositoryID: repo.ID,
		Repository:   repo.Name,
		WorkflowName: "myworkflow",
		WorkflowSha:  "azerty",
		WorkflowRef:  "refs/heads/main",
		Status:       sdk.V2WorkflowRunStatusBuilding,
		RunNumber:    1,
		RunAttempt:   1,
		Started:      time.Now(),
		LastModified: time.Now(),
		WorkflowData: sdk.V2WorkflowRunData{},
		Contexts:     sdk.WorkflowRunContext{},
		Initiator:    &sdk.V2Initiator{UserID: admin.ID},
	}
	require.NoError(t, workflow_v2.InsertRun(context.TODO(), db, &wr))

	wrOld := sdk.V2WorkflowRun{
		ID:           sdk.UUID(),
		ProjectKey:   proj.Key,
		VCSServerID:  vcsServer.ID,
		VCSServer:    vcsServer.Name,
		RepositoryID: repo.ID,
		Repository:   repo.Name,
		WorkflowName: "myworkflow",
		WorkflowSha:  "azerty",
		WorkflowRef:  "refs/heads/main",
		Status:       sdk.V2WorkflowRunStatusBlocked,
		RunNumber:    2,
		RunAttempt:   1,
		Started:      time.Now(),
		LastModified: time.Now(),
		WorkflowData: sdk.V2WorkflowRunData{},
		Contexts:     sdk.WorkflowRunContext{},
		Initiator:    &sdk.V2Initiator{UserID: admin.ID},
		Concurrency: &sdk.V2RunConcurrency{
			WorkflowConcurrency: sdk.WorkflowConcurrency{
				Name:             "main",
				Order:            sdk.ConcurrencyOrderNewestFirst,
				Pool:             3,
				CancelInProgress: false,
			},
			Scope: sdk.V2RunConcurrencyScopeProject,
		},
	}
	require.NoError(t, workflow_v2.InsertRun(context.TODO(), db, &wrOld))
	_, err := db.Exec("UPDATE v2_workflow_run SET last_modified = $1", time.Now().Add(10*time.Hour))
	require.NoError(t, err)

	jobRunOld := sdk.V2WorkflowRunJob{
		ID:            sdk.UUID(),
		JobID:         sdk.RandomString(10),
		WorkflowRunID: wr.ID,
		ProjectKey:    wr.ProjectKey,
		VCSServer:     wr.VCSServer,
		Repository:    wr.Repository,
		WorkflowName:  wr.WorkflowName,
		RunNumber:     wr.RunNumber,
		RunAttempt:    wr.RunAttempt,
		Status:        sdk.StatusBlocked,
		Queued:        time.Now(),
		Job:           sdk.V2Job{},
		Concurrency: &sdk.V2RunConcurrency{
			WorkflowConcurrency: sdk.WorkflowConcurrency{
				Name:             "main",
				Order:            sdk.ConcurrencyOrderNewestFirst,
				Pool:             3,
				CancelInProgress: false,
			},
			Scope: sdk.V2RunConcurrencyScopeProject,
		},
		Initiator: *wr.Initiator,
	}
	jobRunOld2 := sdk.V2WorkflowRunJob{
		ID:            sdk.UUID(),
		JobID:         sdk.RandomString(10),
		WorkflowRunID: wr.ID,
		ProjectKey:    wr.ProjectKey,
		VCSServer:     wr.VCSServer,
		Repository:    wr.Repository,
		WorkflowName:  wr.WorkflowName,
		RunNumber:     wr.RunNumber,
		RunAttempt:    wr.RunAttempt,
		Status:        sdk.StatusBlocked,
		Queued:        time.Now().Add(30 * time.Second),
		Job:           sdk.V2Job{},
		Concurrency: &sdk.V2RunConcurrency{
			WorkflowConcurrency: sdk.WorkflowConcurrency{
				Name:             "main",
				Order:            sdk.ConcurrencyOrderNewestFirst,
				Pool:             3,
				CancelInProgress: false,
			},
			Scope: sdk.V2RunConcurrencyScopeProject,
		},
		Initiator: *wr.Initiator,
	}
	jobRunNew := sdk.V2WorkflowRunJob{
		ID:            sdk.UUID(),
		JobID:         sdk.RandomString(10),
		WorkflowRunID: wr.ID,
		ProjectKey:    wr.ProjectKey,
		VCSServer:     wr.VCSServer,
		Repository:    wr.Repository,
		WorkflowName:  wr.WorkflowName,
		RunNumber:     wr.RunNumber,
		RunAttempt:    wr.RunAttempt,
		Status:        sdk.StatusBlocked,
		Queued:        time.Now().Add(1 * time.Minute),
		Job:           sdk.V2Job{},
		Concurrency: &sdk.V2RunConcurrency{
			WorkflowConcurrency: sdk.WorkflowConcurrency{
				Name:             "main",
				Order:            sdk.ConcurrencyOrderNewestFirst,
				Pool:             3,
				CancelInProgress: false,
			},
			Scope: sdk.V2RunConcurrencyScopeProject,
		},
		Initiator: *wr.Initiator,
	}

	require.NoError(t, workflow_v2.InsertRunJob(context.TODO(), db, &jobRunNew))
	require.NoError(t, workflow_v2.InsertRunJob(context.TODO(), db, &jobRunOld))
	require.NoError(t, workflow_v2.InsertRunJob(context.TODO(), db, &jobRunOld2))

	objs, _, err := retrieveRunObjectsToUnLocked(context.TODO(), db.DbMap, jobRunNew.ProjectKey, jobRunNew.VCSServer, jobRunNew.Repository, jobRunNew.WorkflowName, *jobRunNew.Concurrency)
	require.NoError(t, err)
	require.Equal(t, 3, len(objs))
	require.Equal(t, wrOld.ID, objs[0].ID)
	require.Equal(t, workflow_v2.ConcurrencyObjectTypeWorkflow, objs[0].Type)
	require.Equal(t, jobRunNew.ID, objs[1].ID)
	require.Equal(t, workflow_v2.ConcurrencyObjectTypeJob, objs[1].Type)
	require.Equal(t, jobRunOld2.ID, objs[2].ID)
	require.Equal(t, workflow_v2.ConcurrencyObjectTypeJob, objs[2].Type)
}

func TestCheckJobProjectConcurrency_DefaultRules(t *testing.T) {
	api, db, _ := newTestAPI(t)

	admin, _ := assets.InsertAdminUser(t, db)
	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))
	vcsServer := assets.InsertTestVCSProject(t, db, proj.ID, "github", "github")
	repo := assets.InsertTestProjectRepository(t, db, proj.Key, vcsServer.ID, sdk.RandomString(10))

	wr := sdk.V2WorkflowRun{
		ID:           sdk.UUID(),
		ProjectKey:   proj.Key,
		VCSServerID:  vcsServer.ID,
		VCSServer:    vcsServer.Name,
		RepositoryID: repo.ID,
		Repository:   repo.Name,
		WorkflowName: "myworkflow",
		WorkflowSha:  "azerty",
		WorkflowRef:  "refs/heads/main",
		Status:       sdk.V2WorkflowRunStatusBuilding,
		RunNumber:    1,
		RunAttempt:   1,
		Started:      time.Now(),
		LastModified: time.Now(),
		WorkflowData: sdk.V2WorkflowRunData{},
		Contexts:     sdk.WorkflowRunContext{},
		Initiator:    &sdk.V2Initiator{UserID: admin.ID},
	}
	require.NoError(t, workflow_v2.InsertRun(context.TODO(), db, &wr))

	jobRunOld := sdk.V2WorkflowRunJob{
		ID:            sdk.UUID(),
		JobID:         sdk.RandomString(10),
		WorkflowRunID: wr.ID,
		ProjectKey:    wr.ProjectKey,
		VCSServer:     wr.VCSServer,
		Repository:    wr.Repository,
		WorkflowName:  wr.WorkflowName,
		RunNumber:     wr.RunNumber,
		RunAttempt:    wr.RunAttempt,
		Status:        sdk.StatusBlocked,
		Queued:        time.Now(),
		Job:           sdk.V2Job{},
		Concurrency: &sdk.V2RunConcurrency{
			WorkflowConcurrency: sdk.WorkflowConcurrency{
				Name:             "main",
				Order:            sdk.ConcurrencyOrderOldestFirst,
				Pool:             3,
				CancelInProgress: true,
			},
			Scope: sdk.V2RunConcurrencyScopeProject,
		},
		Initiator: *wr.Initiator,
	}
	jobRunNew := sdk.V2WorkflowRunJob{
		ID:            sdk.UUID(),
		JobID:         sdk.RandomString(10),
		WorkflowRunID: wr.ID,
		ProjectKey:    wr.ProjectKey,
		VCSServer:     wr.VCSServer,
		Repository:    wr.Repository,
		WorkflowName:  wr.WorkflowName,
		RunNumber:     wr.RunNumber,
		RunAttempt:    wr.RunAttempt,
		Status:        sdk.StatusBlocked,
		Queued:        time.Now().Add(1 * time.Minute),
		Job:           sdk.V2Job{},
		Concurrency: &sdk.V2RunConcurrency{
			WorkflowConcurrency: sdk.WorkflowConcurrency{
				Name:             "main",
				Order:            sdk.ConcurrencyOrderNewestFirst,
				Pool:             10,
				CancelInProgress: false,
			},
			Scope: sdk.V2RunConcurrencyScopeProject,
		},
		Initiator: *wr.Initiator,
	}
	require.NoError(t, workflow_v2.InsertRunJob(context.TODO(), db, &jobRunNew))
	require.NoError(t, workflow_v2.InsertRunJob(context.TODO(), db, &jobRunOld))

	rule, nbBuilding, nbBlocking, err := checkProjectScopedConcurrency(context.TODO(), db, wr.ProjectKey, jobRunNew.Concurrency.WorkflowConcurrency)
	require.NoError(t, err)

	require.Equal(t, int64(0), nbBuilding)
	require.Equal(t, int64(2), nbBlocking)

	require.Equal(t, sdk.ConcurrencyOrderOldestFirst, rule.Order)
	require.Equal(t, int64(3), rule.Pool)
	require.False(t, rule.CancelInProgress)
}

func TestRetrieveRunJobToUnlocked_CancelInProgress(t *testing.T) {
	api, db, _ := newTestAPI(t)

	admin, _ := assets.InsertAdminUser(t, db)
	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))
	vcsServer := assets.InsertTestVCSProject(t, db, proj.ID, "github", "github")
	repo := assets.InsertTestProjectRepository(t, db, proj.Key, vcsServer.ID, sdk.RandomString(10))

	wr := sdk.V2WorkflowRun{
		ID:           sdk.UUID(),
		ProjectKey:   proj.Key,
		VCSServerID:  vcsServer.ID,
		VCSServer:    vcsServer.Name,
		RepositoryID: repo.ID,
		Repository:   repo.Name,
		WorkflowName: "myworkflow",
		WorkflowSha:  "azerty",
		WorkflowRef:  "refs/heads/main",
		Status:       sdk.V2WorkflowRunStatusBuilding,
		RunNumber:    1,
		RunAttempt:   1,
		Started:      time.Now(),
		LastModified: time.Now(),
		WorkflowData: sdk.V2WorkflowRunData{},
		Contexts:     sdk.WorkflowRunContext{},
		Initiator:    &sdk.V2Initiator{UserID: admin.ID},
	}
	require.NoError(t, workflow_v2.InsertRun(context.TODO(), db, &wr))

	wrOld := sdk.V2WorkflowRun{
		ID:           sdk.UUID(),
		ProjectKey:   proj.Key,
		VCSServerID:  vcsServer.ID,
		VCSServer:    vcsServer.Name,
		RepositoryID: repo.ID,
		Repository:   repo.Name,
		WorkflowName: "myworkflow",
		WorkflowSha:  "azerty",
		WorkflowRef:  "refs/heads/main",
		Status:       sdk.V2WorkflowRunStatusBlocked,
		RunNumber:    2,
		RunAttempt:   1,
		Started:      time.Now(),
		LastModified: time.Now(),
		WorkflowData: sdk.V2WorkflowRunData{},
		Contexts:     sdk.WorkflowRunContext{},
		Initiator:    &sdk.V2Initiator{UserID: admin.ID},
		Concurrency: &sdk.V2RunConcurrency{
			WorkflowConcurrency: sdk.WorkflowConcurrency{
				Name:             "main",
				Order:            sdk.ConcurrencyOrderOldestFirst,
				Pool:             2,
				CancelInProgress: true,
			},
			Scope: sdk.V2RunConcurrencyScopeProject,
		},
	}
	require.NoError(t, workflow_v2.InsertRun(context.TODO(), db, &wrOld))

	jobRun1 := sdk.V2WorkflowRunJob{
		ID:            sdk.UUID(),
		JobID:         sdk.RandomString(10),
		WorkflowRunID: wr.ID,
		ProjectKey:    wr.ProjectKey,
		VCSServer:     wr.VCSServer,
		Repository:    wr.Repository,
		WorkflowName:  wr.WorkflowName,
		RunNumber:     wr.RunNumber,
		RunAttempt:    wr.RunAttempt,
		Status:        sdk.StatusBlocked,
		Queued:        time.Now(),
		Job:           sdk.V2Job{},
		Concurrency: &sdk.V2RunConcurrency{
			WorkflowConcurrency: sdk.WorkflowConcurrency{
				Name:             "main",
				Order:            sdk.ConcurrencyOrderOldestFirst,
				Pool:             2,
				CancelInProgress: true,
			},
			Scope: sdk.V2RunConcurrencyScopeProject,
		},
		Initiator: *wr.Initiator,
	}
	jobRun2 := sdk.V2WorkflowRunJob{
		ID:            sdk.UUID(),
		JobID:         sdk.RandomString(10),
		WorkflowRunID: wr.ID,
		ProjectKey:    wr.ProjectKey,
		VCSServer:     wr.VCSServer,
		Repository:    wr.Repository,
		WorkflowName:  wr.WorkflowName,
		RunNumber:     wr.RunNumber,
		RunAttempt:    wr.RunAttempt,
		Status:        sdk.StatusBlocked,
		Queued:        time.Now().Add(1 * time.Minute),
		Job:           sdk.V2Job{},
		Concurrency: &sdk.V2RunConcurrency{
			WorkflowConcurrency: sdk.WorkflowConcurrency{
				Name:             "main",
				Order:            sdk.ConcurrencyOrderOldestFirst,
				Pool:             2,
				CancelInProgress: true,
			},
			Scope: sdk.V2RunConcurrencyScopeProject,
		},
		Initiator: *wr.Initiator,
	}
	jobRun3 := sdk.V2WorkflowRunJob{
		ID:            sdk.UUID(),
		JobID:         sdk.RandomString(10),
		WorkflowRunID: wr.ID,
		ProjectKey:    wr.ProjectKey,
		VCSServer:     wr.VCSServer,
		Repository:    wr.Repository,
		WorkflowName:  wr.WorkflowName,
		RunNumber:     wr.RunNumber,
		RunAttempt:    wr.RunAttempt,
		Status:        sdk.StatusBlocked,
		Queued:        time.Now().Add(2 * time.Minute),
		Job:           sdk.V2Job{},
		Concurrency: &sdk.V2RunConcurrency{
			WorkflowConcurrency: sdk.WorkflowConcurrency{
				Name:             "main",
				Order:            sdk.ConcurrencyOrderOldestFirst,
				Pool:             2,
				CancelInProgress: true,
			},
			Scope: sdk.V2RunConcurrencyScopeProject,
		},
		Initiator: *wr.Initiator,
	}

	require.NoError(t, workflow_v2.InsertRunJob(context.TODO(), db, &jobRun1))
	require.NoError(t, workflow_v2.InsertRunJob(context.TODO(), db, &jobRun2))
	require.NoError(t, workflow_v2.InsertRunJob(context.TODO(), db, &jobRun3))

	rjs, toCancel, err := retrieveRunObjectsToUnLocked(context.TODO(), db.DbMap, jobRun1.ProjectKey, jobRun1.VCSServer, jobRun1.Repository, jobRun1.WorkflowName, *jobRun1.Concurrency)
	require.NoError(t, err)

	// Check job to unlock
	require.Equal(t, 2, len(rjs))
	var job2, job3 bool
	for _, rj := range rjs {
		if rj.ID == jobRun2.ID {
			job2 = true
		}
		if rj.ID == jobRun3.ID {
			job3 = true
		}
	}
	require.True(t, job2)
	require.True(t, job3)

	// Check job to cancel
	require.Equal(t, 2, len(toCancel))
	var job1, runOld bool
	for _, o := range toCancel {
		if o.ID == wrOld.ID {
			runOld = true
			require.Equal(t, workflow_v2.ConcurrencyObjectTypeWorkflow, o.Type)
		}
		if o.ID == jobRun1.ID {
			job1 = true
			require.Equal(t, workflow_v2.ConcurrencyObjectTypeJob, o.Type)
		}
	}
	require.True(t, job1)
	require.True(t, runOld)

	require.Equal(t, jobRun1.ID, toCancel[0].ID)

}

// A run failed by the engine in the middle of its execution must release the concurrency held by its
// blocked jobs, otherwise the oldest_first queue stays stuck on a job that will never be unlocked.
func TestFailedRunReleasesBlockedJobConcurrency(t *testing.T) {
	api, db, _ := newTestAPI(t)

	_, err := db.Exec("DELETE FROM rbac")
	require.NoError(t, err)
	_, err = db.Exec("DELETE FROM region")
	require.NoError(t, err)

	admin, _ := assets.InsertAdminUser(t, db)
	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))
	vcsServer := assets.InsertTestVCSProject(t, db, proj.ID, "github", "github")
	repo := assets.InsertTestProjectRepository(t, db, proj.Key, vcsServer.ID, sdk.RandomString(10))

	reg := sdk.Region{Name: "build"}
	require.NoError(t, region.Insert(context.TODO(), db, &reg))
	api.Config.Workflow.JobDefaultRegion = reg.Name
	require.NoError(t, rbac.Insert(context.TODO(), db, &sdk.RBAC{
		Name: sdk.RandomString(10),
		RegionProjects: []sdk.RBACRegionProject{{
			RegionID:        reg.ID,
			RBACProjectKeys: []string{proj.Key},
			Role:            sdk.RegionRoleExecute,
		}},
	}))

	concurrency := sdk.V2RunConcurrency{
		WorkflowConcurrency: sdk.WorkflowConcurrency{
			Name:  "mycc",
			Order: sdk.ConcurrencyOrderOldestFirst,
			Pool:  1,
		},
		Scope: sdk.V2RunConcurrencyScopeWorkflow,
	}
	initiator := &sdk.V2Initiator{UserID: admin.ID, User: admin.Initiator(), IsAdminWithMFA: true}

	newRun := func(runNumber int64) sdk.V2WorkflowRun {
		return sdk.V2WorkflowRun{
			ProjectKey:   proj.Key,
			VCSServerID:  vcsServer.ID,
			VCSServer:    vcsServer.Name,
			RepositoryID: repo.ID,
			Repository:   repo.Name,
			WorkflowName: "myworkflow",
			WorkflowSha:  "abcdef",
			WorkflowRef:  "refs/heads/master",
			RunNumber:    runNumber,
			Status:       sdk.V2WorkflowRunStatusBuilding,
			Initiator:    initiator,
			WorkflowData: sdk.V2WorkflowRunData{Workflow: sdk.V2Workflow{
				Name:          "myworkflow",
				Concurrencies: []sdk.WorkflowConcurrency{concurrency.WorkflowConcurrency},
				Jobs: map[string]sdk.V2Job{
					"build": {
						Steps: []sdk.ActionStep{{Run: "echo build"}},
					},
					"test": {
						Steps: []sdk.ActionStep{{Run: "echo test"}},
					},
					"deploy": {
						Concurrency: concurrency.Name,
						Steps:       []sdk.ActionStep{{Run: "echo deploy"}},
					},
					// Matrix computed from an output that build doesn't produce: only checked once publish can be enqueued, after build
					"publish": {
						Needs: []string{"build"},
						Strategy: &sdk.V2JobStrategy{
							Matrix: map[string]interface{}{"region": "${{ needs.build.outputs.regions }}"},
						},
						Steps: []sdk.ActionStep{{Run: "echo publish"}},
					},
				},
			}},
		}
	}

	// Another run holds the concurrency
	holder := newRun(1)
	require.NoError(t, workflow_v2.InsertRun(context.TODO(), db, &holder))
	holderJob := sdk.V2WorkflowRunJob{
		JobID:         "deploy",
		WorkflowRunID: holder.ID,
		ProjectKey:    holder.ProjectKey,
		VCSServer:     holder.VCSServer,
		Repository:    holder.Repository,
		WorkflowName:  holder.WorkflowName,
		RunNumber:     holder.RunNumber,
		RunAttempt:    holder.RunAttempt,
		Status:        sdk.V2WorkflowRunJobStatusBuilding,
		Job:           holder.WorkflowData.Workflow.Jobs["deploy"],
		Concurrency:   &concurrency,
		Initiator:     *initiator,
	}
	require.NoError(t, workflow_v2.InsertRunJob(context.TODO(), db, &holderJob))

	wr := newRun(2)
	require.NoError(t, workflow_v2.InsertRun(context.TODO(), db, &wr))

	// First trigger: build is enqueued, deploy is blocked by the holder, publish waits for build
	require.NoError(t, api.workflowRunV2Trigger(context.TODO(), sdk.V2WorkflowRunEnqueue{RunID: wr.ID, Initiator: *initiator}))

	wrDB, err := workflow_v2.LoadRunByID(context.TODO(), db, wr.ID)
	require.NoError(t, err)
	require.Equal(t, sdk.V2WorkflowRunStatusBuilding, wrDB.Status, "first trigger must not fail the run")

	runJobs, err := workflow_v2.LoadRunJobsByRunID(context.TODO(), db, wr.ID, wr.RunAttempt)
	require.NoError(t, err)
	jobsByID := make(map[string]sdk.V2WorkflowRunJob)
	for _, rj := range runJobs {
		jobsByID[rj.JobID] = rj
	}
	require.Len(t, jobsByID, 3)
	require.Equal(t, sdk.V2WorkflowRunJobStatusWaiting, jobsByID["build"].Status)
	require.Equal(t, sdk.V2WorkflowRunJobStatusWaiting, jobsByID["test"].Status)
	require.Equal(t, sdk.V2WorkflowRunJobStatusBlocked, jobsByID["deploy"].Status)

	// test is taken by a worker
	testJob := jobsByID["test"]
	testJob.Status = sdk.V2WorkflowRunJobStatusBuilding
	testJob.StepsStatus = sdk.JobStepsStatus{
		"step-0": {Conclusion: sdk.V2WorkflowRunJobStatusSuccess, Outcome: sdk.V2WorkflowRunJobStatusSuccess, Started: time.Now(), Ended: time.Now()},
		"step-1": {Conclusion: sdk.V2WorkflowRunJobStatusBuilding, Outcome: sdk.V2WorkflowRunJobStatusBuilding, Started: time.Now()},
	}
	require.NoError(t, workflow_v2.UpdateJobRun(context.TODO(), db, &testJob))

	// build ends: the next trigger enqueues publish and fails the run
	buildJob := jobsByID["build"]
	buildJob.Status = sdk.V2WorkflowRunJobStatusSuccess
	require.NoError(t, workflow_v2.UpdateJobRun(context.TODO(), db, &buildJob))
	require.NoError(t, api.workflowRunV2Trigger(context.TODO(), sdk.V2WorkflowRunEnqueue{RunID: wr.ID, Initiator: *initiator}))

	wrDB, err = workflow_v2.LoadRunByID(context.TODO(), db, wr.ID)
	require.NoError(t, err)
	require.Equal(t, sdk.V2WorkflowRunStatusFail, wrDB.Status, "second trigger must fail the run")

	runInfos, err := workflow_v2.LoadRunInfosByRunID(context.TODO(), db, wr.ID)
	require.NoError(t, err)
	require.Len(t, runInfos, 1)
	require.Contains(t, runInfos[0].Message, "interpolated matrix is not a string slice")

	blockedJobDB, err := workflow_v2.LoadRunJobByID(context.TODO(), db, jobsByID["deploy"].ID)
	require.NoError(t, err)
	assert.Equal(t, sdk.V2WorkflowRunJobStatusStopped, blockedJobDB.Status, "job of a failed run should be stopped")
	require.NotNil(t, blockedJobDB.Ended)
	jobInfos, err := workflow_v2.LoadRunJobInfosByRunJobID(context.TODO(), db, blockedJobDB.ID)
	require.NoError(t, err)
	var stoppedInfo bool
	for _, info := range jobInfos {
		if info.Message == "Job stopped because the workflow run failed" {
			stoppedInfo = true
		}
	}
	assert.True(t, stoppedInfo, "stopped job should have an info message")

	// A running job is stopped too, so that it doesn't keep running outside of the run concurrency
	testJobDB, err := workflow_v2.LoadRunJobByID(context.TODO(), db, testJob.ID)
	require.NoError(t, err)
	assert.Equal(t, sdk.V2WorkflowRunJobStatusStopped, testJobDB.Status, "running job of a failed run should be stopped")
	assert.Equal(t, sdk.V2WorkflowRunJobStatusSuccess, testJobDB.StepsStatus["step-0"].Conclusion, "ended step should be kept")
	assert.Equal(t, sdk.V2WorkflowRunJobStatusStopped, testJobDB.StepsStatus["step-1"].Conclusion, "running step should be stopped")
	assert.False(t, testJobDB.StepsStatus["step-1"].Ended.IsZero())

	// The holder ends: nothing is running anymore on the concurrency
	holderJob.Status = sdk.V2WorkflowRunJobStatusSuccess
	require.NoError(t, workflow_v2.UpdateJobRun(context.TODO(), db, &holderJob))

	wr3 := newRun(3)
	require.NoError(t, workflow_v2.InsertRun(context.TODO(), db, &wr3))
	canRun, _, err := canRunWithConcurrency(context.TODO(), concurrency, db.DbMap, wr3,
		workflow_v2.ConcurrencyObject{ID: sdk.UUID(), Type: workflow_v2.ConcurrencyObjectTypeJob},
		map[string]int64{}, map[string]workflow_v2.ConcurrencyObject{})
	require.NoError(t, err)
	assert.True(t, canRun, "new job should not be blocked by a job of a failed run")

	// The unlock routine must not select a job whose run is already terminated
	toUnlock, _, err := retrieveRunObjectsToUnLocked(context.TODO(), db.DbMap, wr.ProjectKey, wr.VCSServer, wr.Repository, wr.WorkflowName, concurrency)
	require.NoError(t, err)
	for _, o := range toUnlock {
		assert.NotEqual(t, jobsByID["deploy"].ID, o.ID, "job of a failed run selected for unlock")
	}
}

func insertRunWithConcurrency(t *testing.T, db gorpmapper.SqlExecutorWithTx, proj sdk.Project, vcsServer sdk.VCSProject, repo sdk.ProjectRepository, initiator *sdk.V2Initiator, runNumber int64, status sdk.V2WorkflowRunStatus, concurrency sdk.V2RunConcurrency) sdk.V2WorkflowRun {
	wr := sdk.V2WorkflowRun{
		ProjectKey:   proj.Key,
		VCSServerID:  vcsServer.ID,
		VCSServer:    vcsServer.Name,
		RepositoryID: repo.ID,
		Repository:   repo.Name,
		WorkflowName: "myworkflow",
		WorkflowSha:  "abcdef",
		WorkflowRef:  "refs/heads/master",
		RunNumber:    runNumber,
		Status:       status,
		Initiator:    initiator,
		Concurrency:  &concurrency,
		WorkflowData: sdk.V2WorkflowRunData{Workflow: sdk.V2Workflow{
			Name:          "myworkflow",
			Concurrency:   concurrency.Name,
			Concurrencies: []sdk.WorkflowConcurrency{concurrency.WorkflowConcurrency},
			Jobs: map[string]sdk.V2Job{
				"job1": {Steps: []sdk.ActionStep{{Run: "echo job1"}}},
			},
		}},
	}
	require.NoError(t, workflow_v2.InsertRun(context.TODO(), db, &wr))
	// Newest objects are selected on last_modified
	time.Sleep(10 * time.Millisecond)
	return wr
}

func cancelInProgressConcurrency(scope sdk.V2RunJobConcurrencyScope, name string) sdk.V2RunConcurrency {
	return sdk.V2RunConcurrency{
		WorkflowConcurrency: sdk.WorkflowConcurrency{
			Name:             name,
			Order:            sdk.ConcurrencyOrderOldestFirst,
			Pool:             1,
			CancelInProgress: true,
		},
		Scope: scope,
	}
}

// A new run must cancel the in-progress run and the older runs blocked waiting for its cancellation.
func TestConcurrencyCancelInProgress_NewRunCancelsOlderBlockedRun(t *testing.T) {
	api, db, _ := newTestAPI(t)

	admin, _ := assets.InsertAdminUser(t, db)
	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))
	vcsServer := assets.InsertTestVCSProject(t, db, proj.ID, "github", "github")
	repo := assets.InsertTestProjectRepository(t, db, proj.Key, vcsServer.ID, sdk.RandomString(10))
	initiator := &sdk.V2Initiator{UserID: admin.ID, User: admin.Initiator(), IsAdminWithMFA: true}
	concurrency := cancelInProgressConcurrency(sdk.V2RunConcurrencyScopeWorkflow, "mycc")

	// A is being cancelled for B, B waits for the cancellation of A
	runA := insertRunWithConcurrency(t, db, *proj, *vcsServer, *repo, initiator, 1, sdk.V2WorkflowRunStatusBuilding, concurrency)
	runB := insertRunWithConcurrency(t, db, *proj, *vcsServer, *repo, initiator, 2, sdk.V2WorkflowRunStatusBlocked, concurrency)
	runC := insertRunWithConcurrency(t, db, *proj, *vcsServer, *repo, initiator, 3, sdk.V2WorkflowRunStatusCrafting, concurrency)

	toCancel := make(map[string]workflow_v2.ConcurrencyObject)
	_, err := manageWorkflowConcurrency(context.TODO(), db.DbMap, &runC, map[string]int64{}, toCancel)
	require.NoError(t, err)

	assert.Contains(t, toCancel, runA.ID, "in-progress run must be cancelled")
	assert.Contains(t, toCancel, runB.ID, "older blocked run must be cancelled")
	assert.Equal(t, sdk.V2WorkflowRunStatusBlocked, runC.Status, "new run must wait for the cancellation of the in-progress run")
}

// With cancel-in-progress, a blocked run waiting for an in-progress run to be cancelled must stay blocked.
func TestConcurrencyCancelInProgress_UnlockKeepsNewestBlockedWhilePoolIsTaken(t *testing.T) {
	api, db, _ := newTestAPI(t)

	admin, _ := assets.InsertAdminUser(t, db)
	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))
	vcsServer := assets.InsertTestVCSProject(t, db, proj.ID, "github", "github")
	repo := assets.InsertTestProjectRepository(t, db, proj.Key, vcsServer.ID, sdk.RandomString(10))
	initiator := &sdk.V2Initiator{UserID: admin.ID, User: admin.Initiator(), IsAdminWithMFA: true}
	concurrency := cancelInProgressConcurrency(sdk.V2RunConcurrencyScopeWorkflow, "mycc")

	// Run being cancelled, still building
	_ = insertRunWithConcurrency(t, db, *proj, *vcsServer, *repo, initiator, 1, sdk.V2WorkflowRunStatusBuilding, concurrency)
	newest := insertRunWithConcurrency(t, db, *proj, *vcsServer, *repo, initiator, 2, sdk.V2WorkflowRunStatusBlocked, concurrency)

	require.NoError(t, api.workflowRunV2Trigger(context.TODO(), sdk.V2WorkflowRunEnqueue{RunID: newest.ID, Initiator: *initiator}))

	newestDB, err := workflow_v2.LoadRunByID(context.TODO(), db, newest.ID)
	require.NoError(t, err)
	assert.Equal(t, sdk.V2WorkflowRunStatusBlocked, newestDB.Status, "newest run must wait for the in-progress run to end")

	toUnlock, toCancel, err := retrieveRunObjectsToUnLocked(context.TODO(), db.DbMap, proj.Key, vcsServer.Name, repo.Name, "myworkflow", concurrency)
	require.NoError(t, err)
	assert.Empty(t, toUnlock)
	assert.Empty(t, toCancel, "unlocking must not cancel anything")
}

// More in-progress runs than the pool allows must not break the unlock of a cancel-in-progress concurrency.
func TestConcurrencyCancelInProgress_UnlockWithMoreBuildingThanPool(t *testing.T) {
	api, db, _ := newTestAPI(t)

	admin, _ := assets.InsertAdminUser(t, db)
	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))
	vcsServer := assets.InsertTestVCSProject(t, db, proj.ID, "github", "github")
	repo := assets.InsertTestProjectRepository(t, db, proj.Key, vcsServer.ID, sdk.RandomString(10))
	initiator := &sdk.V2Initiator{UserID: admin.ID, User: admin.Initiator(), IsAdminWithMFA: true}
	concurrency := cancelInProgressConcurrency(sdk.V2RunConcurrencyScopeWorkflow, "mycc")

	_ = insertRunWithConcurrency(t, db, *proj, *vcsServer, *repo, initiator, 1, sdk.V2WorkflowRunStatusBuilding, concurrency)
	_ = insertRunWithConcurrency(t, db, *proj, *vcsServer, *repo, initiator, 2, sdk.V2WorkflowRunStatusBuilding, concurrency)
	_ = insertRunWithConcurrency(t, db, *proj, *vcsServer, *repo, initiator, 3, sdk.V2WorkflowRunStatusBlocked, concurrency)

	var toUnlock []workflow_v2.ConcurrencyObject
	var err error
	require.NotPanics(t, func() {
		toUnlock, _, err = retrieveRunObjectsToUnLocked(context.TODO(), db.DbMap, proj.Key, vcsServer.Name, repo.Name, "myworkflow", concurrency)
	})
	require.NoError(t, err)
	assert.Empty(t, toUnlock)
}

// Triggering a run must not cancel its job blocked by a cancel-in-progress concurrency while the pool is taken.
func TestConcurrencyCancelInProgress_RunTriggerDoesNotCancelBlockedJob(t *testing.T) {
	api, db, _ := newTestAPI(t)

	admin, _ := assets.InsertAdminUser(t, db)
	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))
	vcsServer := assets.InsertTestVCSProject(t, db, proj.ID, "github", "github")
	repo := assets.InsertTestProjectRepository(t, db, proj.Key, vcsServer.ID, sdk.RandomString(10))
	initiator := &sdk.V2Initiator{UserID: admin.ID, User: admin.Initiator(), IsAdminWithMFA: true}

	pc := sdk.ProjectConcurrency{ProjectKey: proj.Key, Name: "deploy", Pool: 1, Order: sdk.ConcurrencyOrderOldestFirst, CancelInProgress: true}
	require.NoError(t, project.InsertConcurrency(context.TODO(), db, &pc))
	concurrency := sdk.V2RunConcurrency{WorkflowConcurrency: pc.ToWorkflowConcurrency(), Scope: sdk.V2RunConcurrencyScopeProject}

	// Another workflow holds the concurrency and is being cancelled for the deploy job
	_ = insertRunWithConcurrency(t, db, *proj, *vcsServer, *repo, initiator, 1, sdk.V2WorkflowRunStatusBuilding, concurrency)

	wr := sdk.V2WorkflowRun{
		ProjectKey:   proj.Key,
		VCSServerID:  vcsServer.ID,
		VCSServer:    vcsServer.Name,
		RepositoryID: repo.ID,
		Repository:   repo.Name,
		WorkflowName: "otherworkflow",
		WorkflowSha:  "abcdef",
		WorkflowRef:  "refs/heads/master",
		RunNumber:    1,
		Status:       sdk.V2WorkflowRunStatusBuilding,
		Initiator:    initiator,
		WorkflowData: sdk.V2WorkflowRunData{Workflow: sdk.V2Workflow{
			Name: "otherworkflow",
			Jobs: map[string]sdk.V2Job{
				"deploy": {Concurrency: pc.Name, Steps: []sdk.ActionStep{{Run: "echo deploy"}}},
			},
		}},
	}
	require.NoError(t, workflow_v2.InsertRun(context.TODO(), db, &wr))
	blockedJob := sdk.V2WorkflowRunJob{
		JobID:         "deploy",
		WorkflowRunID: wr.ID,
		ProjectKey:    wr.ProjectKey,
		VCSServer:     wr.VCSServer,
		Repository:    wr.Repository,
		WorkflowName:  wr.WorkflowName,
		RunNumber:     wr.RunNumber,
		RunAttempt:    wr.RunAttempt,
		Status:        sdk.V2WorkflowRunJobStatusBlocked,
		Queued:        time.Now(),
		Job:           wr.WorkflowData.Workflow.Jobs["deploy"],
		Concurrency:   &concurrency,
		Initiator:     *initiator,
	}
	require.NoError(t, workflow_v2.InsertRunJob(context.TODO(), db, &blockedJob))

	require.NoError(t, api.workflowRunV2Trigger(context.TODO(), sdk.V2WorkflowRunEnqueue{RunID: wr.ID, Initiator: *initiator}))

	blockedJobDB, err := workflow_v2.LoadRunJobByID(context.TODO(), db, blockedJob.ID)
	require.NoError(t, err)
	assert.Equal(t, sdk.V2WorkflowRunJobStatusBlocked, blockedJobDB.Status, "blocked job must wait for the in-progress run to end")
}

// A job must only wait for the cancellation of a workflow holding its own concurrency.
func TestManageJobConcurrency_WaitsOnlyForCancellationOnSameConcurrency(t *testing.T) {
	api, db, _ := newTestAPI(t)

	admin, _ := assets.InsertAdminUser(t, db)
	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))
	vcsServer := assets.InsertTestVCSProject(t, db, proj.ID, "github", "github")
	repo := assets.InsertTestProjectRepository(t, db, proj.Key, vcsServer.ID, sdk.RandomString(10))
	initiator := &sdk.V2Initiator{UserID: admin.ID, User: admin.Initiator(), IsAdminWithMFA: true}

	pcDeploy := sdk.ProjectConcurrency{ProjectKey: proj.Key, Name: "deploy", Pool: 1, Order: sdk.ConcurrencyOrderOldestFirst, CancelInProgress: true}
	require.NoError(t, project.InsertConcurrency(context.TODO(), db, &pcDeploy))
	pcLint := sdk.ProjectConcurrency{ProjectKey: proj.Key, Name: "lint", Pool: 1, Order: sdk.ConcurrencyOrderOldestFirst}
	require.NoError(t, project.InsertConcurrency(context.TODO(), db, &pcLint))
	deployConcurrency := sdk.V2RunConcurrency{WorkflowConcurrency: pcDeploy.ToWorkflowConcurrency(), Scope: sdk.V2RunConcurrencyScopeProject}
	lintConcurrency := sdk.V2RunConcurrency{WorkflowConcurrency: pcLint.ToWorkflowConcurrency(), Scope: sdk.V2RunConcurrencyScopeProject}

	// Another workflow holds the deploy concurrency, nothing holds the lint one
	holder := insertRunWithConcurrency(t, db, *proj, *vcsServer, *repo, initiator, 1, sdk.V2WorkflowRunStatusBuilding, deployConcurrency)

	wr := sdk.V2WorkflowRun{ID: sdk.UUID(), ProjectKey: proj.Key, VCSServer: vcsServer.Name, Repository: repo.Name, WorkflowName: "otherworkflow", RunNumber: 1, RunAttempt: 1}
	concurrenciesDef := map[string]sdk.V2RunConcurrency{"deploy": deployConcurrency, "lint": lintConcurrency}
	unlockedCount := make(map[string]int64)
	toCancel := make(map[string]workflow_v2.ConcurrencyObject)

	deployJob := sdk.V2WorkflowRunJob{ID: sdk.UUID(), JobID: "deploy", WorkflowRunID: wr.ID, Status: sdk.V2WorkflowRunJobStatusWaiting, Job: sdk.V2Job{Concurrency: pcDeploy.Name}}
	_, err := manageJobConcurrency(context.TODO(), db.DbMap, wr, "deploy", &deployJob, concurrenciesDef, unlockedCount, toCancel)
	require.NoError(t, err)
	require.Contains(t, toCancel, holder.ID)
	require.Equal(t, sdk.V2WorkflowRunJobStatusBlocked, deployJob.Status, "deploy job must wait for the cancellation of the holder")

	lintJob := sdk.V2WorkflowRunJob{ID: sdk.UUID(), JobID: "lint", WorkflowRunID: wr.ID, Status: sdk.V2WorkflowRunJobStatusWaiting, Job: sdk.V2Job{Concurrency: pcLint.Name}}
	_, err = manageJobConcurrency(context.TODO(), db.DbMap, wr, "lint", &lintJob, concurrenciesDef, unlockedCount, toCancel)
	require.NoError(t, err)
	assert.Equal(t, sdk.V2WorkflowRunJobStatusWaiting, lintJob.Status, "lint job doesn't depend on the cancelled workflow")
}

// A blocked run superseded by a newer one is cancelled by the arrival of the newer one, through the engine queue.
func TestConcurrencyCancelInProgress_SupersededBlockedRunIsCancelled(t *testing.T) {
	api, db, _ := newTestAPI(t)
	api.workflowRunTriggerChan = make(chan sdk.V2WorkflowRunEnqueue, 10)

	admin, _ := assets.InsertAdminUser(t, db)
	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))
	vcsServer := assets.InsertTestVCSProject(t, db, proj.ID, "github", "github")
	repo := assets.InsertTestProjectRepository(t, db, proj.Key, vcsServer.ID, sdk.RandomString(10))
	initiator := &sdk.V2Initiator{UserID: admin.ID, User: admin.Initiator(), IsAdminWithMFA: true}
	concurrency := cancelInProgressConcurrency(sdk.V2RunConcurrencyScopeWorkflow, "mycc")

	runA := insertRunWithConcurrency(t, db, *proj, *vcsServer, *repo, initiator, 1, sdk.V2WorkflowRunStatusBuilding, concurrency)
	runB := insertRunWithConcurrency(t, db, *proj, *vcsServer, *repo, initiator, 2, sdk.V2WorkflowRunStatusBlocked, concurrency)
	runC := insertRunWithConcurrency(t, db, *proj, *vcsServer, *repo, initiator, 3, sdk.V2WorkflowRunStatusCrafting, concurrency)

	// Arrival of C, as done when crafting it
	toCancel := make(map[string]workflow_v2.ConcurrencyObject)
	_, err := manageWorkflowConcurrency(context.TODO(), db.DbMap, &runC, map[string]int64{}, toCancel)
	require.NoError(t, err)
	tx, err := db.Begin()
	require.NoError(t, err)
	require.NoError(t, api.cancelRunObjects(context.TODO(), tx, toCancel))
	require.NoError(t, tx.Commit())

	// Both A and B are sent to the engine for cancellation
	cancelled := make(map[string]sdk.V2WorkflowRunEnqueue)
	for i := 0; i < 2; i++ {
		select {
		case e := <-api.workflowRunTriggerChan:
			cancelled[e.RunID] = e
		case <-time.After(5 * time.Second):
			t.Fatal("expected two runs enqueued for cancellation")
		}
	}
	require.Contains(t, cancelled, runA.ID)
	require.Contains(t, cancelled, runB.ID)
	require.Equal(t, sdk.V2WorkflowRunStatusCancelled, cancelled[runB.ID].Status)

	// The engine pass on the blocked run terminates it instead of trying to unlock it
	require.NoError(t, api.workflowRunV2Trigger(context.TODO(), cancelled[runB.ID]))
	runBDB, err := workflow_v2.LoadRunByID(context.TODO(), db, runB.ID)
	require.NoError(t, err)
	assert.Equal(t, sdk.V2WorkflowRunStatusCancelled, runBDB.Status)
	infos, err := workflow_v2.LoadRunInfosByRunID(context.TODO(), db, runB.ID)
	require.NoError(t, err)
	require.Len(t, infos, 1)
	assert.Contains(t, infos[0].Message, "Workflow cancelled due to concurrency")
}
