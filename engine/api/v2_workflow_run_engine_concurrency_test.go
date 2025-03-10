package api

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/engine/api/workflow_v2"
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
				Pool:             2,
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
		Queued:        time.Now().Add(1 * time.Second),
		Job:           sdk.V2Job{},
		Concurrency: &sdk.V2RunConcurrency{
			WorkflowConcurrency: sdk.WorkflowConcurrency{
				Name:             "main",
				Order:            sdk.ConcurrencyOrderOldestFirst,
				Pool:             2,
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
				Pool:             2,
				CancelInProgress: false,
			},
			Scope: sdk.V2RunConcurrencyScopeWorkflow,
		},
		Initiator: *wr.Initiator,
	}

	require.NoError(t, workflow_v2.InsertRunJob(context.TODO(), db, &jobRunNew))
	require.NoError(t, workflow_v2.InsertRunJob(context.TODO(), db, &jobRunOld))
	require.NoError(t, workflow_v2.InsertRunJob(context.TODO(), db, &jobRunOld2))

	rjs, _, err := retrieveRunJobToUnLocked(context.TODO(), db.DbMap, jobRunNew)
	require.NoError(t, err)
	require.Equal(t, 2, len(rjs))
	require.Equal(t, jobRunOld.ID, rjs[0].ID)
	require.Equal(t, jobRunOld2.ID, rjs[1].ID)
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
				Pool:             2,
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
				Pool:             2,
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
				Pool:             2,
				CancelInProgress: false,
			},
			Scope: sdk.V2RunConcurrencyScopeWorkflow,
		},
		Initiator: *wr.Initiator,
	}

	require.NoError(t, workflow_v2.InsertRunJob(context.TODO(), db, &jobRunNew))
	require.NoError(t, workflow_v2.InsertRunJob(context.TODO(), db, &jobRunOld))
	require.NoError(t, workflow_v2.InsertRunJob(context.TODO(), db, &jobRunOld2))

	rjs, _, err := retrieveRunJobToUnLocked(context.TODO(), db.DbMap, jobRunNew)
	require.NoError(t, err)
	require.Equal(t, 2, len(rjs))
	require.Equal(t, jobRunNew.ID, rjs[0].ID)
	require.Equal(t, jobRunOld2.ID, rjs[1].ID)
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
				Pool:             2,
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
				Pool:             2,
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
				Pool:             2,
				CancelInProgress: false,
			},
			Scope: sdk.V2RunConcurrencyScopeProject,
		},
		Initiator: *wr.Initiator,
	}

	require.NoError(t, workflow_v2.InsertRunJob(context.TODO(), db, &jobRunNew))
	require.NoError(t, workflow_v2.InsertRunJob(context.TODO(), db, &jobRunOld))
	require.NoError(t, workflow_v2.InsertRunJob(context.TODO(), db, &jobRunOld2))

	rjs, _, err := retrieveRunJobToUnLocked(context.TODO(), db.DbMap, jobRunNew)
	require.NoError(t, err)
	require.Equal(t, 2, len(rjs))
	require.Equal(t, jobRunOld.ID, rjs[0].ID)
	require.Equal(t, jobRunOld2.ID, rjs[1].ID)
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
				Pool:             2,
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
				Pool:             2,
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
				Pool:             2,
				CancelInProgress: false,
			},
			Scope: sdk.V2RunConcurrencyScopeProject,
		},
		Initiator: *wr.Initiator,
	}

	require.NoError(t, workflow_v2.InsertRunJob(context.TODO(), db, &jobRunNew))
	require.NoError(t, workflow_v2.InsertRunJob(context.TODO(), db, &jobRunOld))
	require.NoError(t, workflow_v2.InsertRunJob(context.TODO(), db, &jobRunOld2))

	rjs, _, err := retrieveRunJobToUnLocked(context.TODO(), db.DbMap, jobRunNew)
	require.NoError(t, err)
	require.Equal(t, 2, len(rjs))
	require.Equal(t, jobRunNew.ID, rjs[0].ID)
	require.Equal(t, jobRunOld2.ID, rjs[1].ID)
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

	rjs, toCancel, err := retrieveRunJobToUnLocked(context.TODO(), db.DbMap, jobRun1)
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
	require.Equal(t, 1, len(toCancel))
	require.Equal(t, jobRun1.ID, toCancel[0])
}
