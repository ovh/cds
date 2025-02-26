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

func TestRetrieveRunJobToUnlocked_OldestFirst(t *testing.T) {
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
		Concurrency: &sdk.V2RunJobConcurrency{
			WorkflowConcurrency: sdk.WorkflowConcurrency{
				Name:             "main",
				Order:            sdk.ConcurrencyOrderOldestFirst,
				Pool:             1,
				CancelInProgress: false,
			},
			Scope: sdk.V2RunJobConcurrencyScopeWorkflow,
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
		Concurrency: &sdk.V2RunJobConcurrency{
			WorkflowConcurrency: sdk.WorkflowConcurrency{
				Name:             "main",
				Order:            sdk.ConcurrencyOrderOldestFirst,
				Pool:             1,
				CancelInProgress: false,
			},
			Scope: sdk.V2RunJobConcurrencyScopeWorkflow,
		},
		Initiator: *wr.Initiator,
	}

	require.NoError(t, workflow_v2.InsertRunJob(context.TODO(), db, &jobRunNew))
	require.NoError(t, workflow_v2.InsertRunJob(context.TODO(), db, &jobRunOld))

	rj, err := retrieveRunJobToUnLocked(context.TODO(), db.DbMap, jobRunNew)
	require.NoError(t, err)
	require.Equal(t, jobRunOld.ID, rj.ID)
}

func TestRetrieveRunJobToUnlocked_NewestFirst(t *testing.T) {
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
		Concurrency: &sdk.V2RunJobConcurrency{
			WorkflowConcurrency: sdk.WorkflowConcurrency{
				Name:             "main",
				Order:            sdk.ConcurrencyOrderNewestFirst,
				Pool:             1,
				CancelInProgress: false,
			},
			Scope: sdk.V2RunJobConcurrencyScopeWorkflow,
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
		Concurrency: &sdk.V2RunJobConcurrency{
			WorkflowConcurrency: sdk.WorkflowConcurrency{
				Name:             "main",
				Order:            sdk.ConcurrencyOrderNewestFirst,
				Pool:             1,
				CancelInProgress: false,
			},
			Scope: sdk.V2RunJobConcurrencyScopeWorkflow,
		},
		Initiator: *wr.Initiator,
	}

	require.NoError(t, workflow_v2.InsertRunJob(context.TODO(), db, &jobRunNew))
	require.NoError(t, workflow_v2.InsertRunJob(context.TODO(), db, &jobRunOld))

	rj, err := retrieveRunJobToUnLocked(context.TODO(), db.DbMap, jobRunNew)
	require.NoError(t, err)
	require.Equal(t, jobRunNew.ID, rj.ID)
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
		Concurrency: &sdk.V2RunJobConcurrency{
			WorkflowConcurrency: sdk.WorkflowConcurrency{
				Name:             "main",
				Order:            sdk.ConcurrencyOrderOldestFirst,
				Pool:             3,
				CancelInProgress: true,
			},
			Scope: sdk.V2RunJobConcurrencyScopeWorkflow,
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
		Concurrency: &sdk.V2RunJobConcurrency{
			WorkflowConcurrency: sdk.WorkflowConcurrency{
				Name:             "main",
				Order:            sdk.ConcurrencyOrderNewestFirst,
				Pool:             10,
				CancelInProgress: false,
			},
			Scope: sdk.V2RunJobConcurrencyScopeWorkflow,
		},
		Initiator: *wr.Initiator,
	}
	require.NoError(t, workflow_v2.InsertRunJob(context.TODO(), db, &jobRunNew))
	require.NoError(t, workflow_v2.InsertRunJob(context.TODO(), db, &jobRunOld))

	rule, nbBuilding, nbBlocking, err := checkJobWorkflowConcurrency(context.TODO(), db, wr.ProjectKey, wr.VCSServer, wr.Repository, wr.WorkflowName, jobRunNew.Concurrency.WorkflowConcurrency)
	require.NoError(t, err)

	require.Equal(t, int64(0), nbBuilding)
	require.Equal(t, int64(2), nbBlocking)

	require.Equal(t, sdk.ConcurrencyOrderOldestFirst, rule.Order)
	require.Equal(t, int64(3), rule.Pool)
	require.False(t, rule.CancelInProgress)
}
