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
