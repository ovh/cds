package worker_v2_test

import (
	"context"
	"testing"
	"time"

	"github.com/ovh/cds/engine/api/bootstrap"
	"github.com/ovh/cds/engine/api/hatchery"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/engine/api/worker_v2"
	"github.com/ovh/cds/engine/api/workflow_v2"
	"github.com/ovh/cds/sdk"
	"github.com/stretchr/testify/require"
)

func TestDeleteDisabledWorkers(t *testing.T) {
	ctx := context.TODO()
	db, cache := test.SetupPG(t, bootstrap.InitiliazeDB)

	db.Exec("DELETE FROM v2_worker")
	db.Exec("DELETE FROM v2_workflow_run_job")

	admin, _ := assets.InsertAdminUser(t, db)
	proj := assets.InsertTestProject(t, db, cache, sdk.RandomString(10), sdk.RandomString(10))
	vcsServer := assets.InsertTestVCSProject(t, db, proj.ID, "github", "github")
	repo := assets.InsertTestProjectRepository(t, db, proj.Key, vcsServer.ID, sdk.RandomString(10))
	wr := sdk.V2WorkflowRun{
		ProjectKey:   proj.Key,
		VCSServerID:  vcsServer.ID,
		VCSServer:    vcsServer.Name,
		RepositoryID: repo.ID,
		Repository:   repo.Name,
		WorkflowName: sdk.RandomString(10),
		WorkflowSha:  "123",
		WorkflowRef:  "master",
		RunAttempt:   0,
		RunNumber:    1,
		Started:      time.Now(),
		LastModified: time.Now(),
		Status:       sdk.StatusBuilding,
		UserID:       admin.ID,
		Username:     admin.Username,
		RunEvent:     sdk.V2WorkflowRunEvent{},
		WorkflowData: sdk.V2WorkflowRunData{Workflow: sdk.V2Workflow{
			Jobs: map[string]sdk.V2Job{
				"job1": {},
				"job2": {
					Needs: []string{"job1"},
				},
			},
		}},
	}
	require.NoError(t, workflow_v2.InsertRun(context.Background(), db, &wr))

	hatch := sdk.Hatchery{
		ModelType: "docker",
		Name:      sdk.RandomString(10),
	}
	require.NoError(t, hatchery.Insert(ctx, db, &hatch))

	wrj := sdk.V2WorkflowRunJob{
		Job:           sdk.V2Job{},
		WorkflowRunID: wr.ID,
		UserID:        admin.ID,
		Username:      admin.Username,
		ProjectKey:    wr.ProjectKey,
		JobID:         sdk.RandomString(10),
		Status:        sdk.StatusBuilding,
	}
	require.NoError(t, workflow_v2.InsertRunJob(context.TODO(), db, &wrj))

	work := sdk.V2Worker{
		Status:       sdk.StatusDisabled,
		JobRunID:     wrj.ID,
		HatcheryName: hatch.Name,
		HatcheryID:   hatch.ID,
		Name:         sdk.RandomString(10),
		LastBeat:     time.Now().Add(-20 * time.Minute),
	}
	require.NoError(t, worker_v2.Insert(ctx, db, &work))

	wrj2 := sdk.V2WorkflowRunJob{
		Job:           sdk.V2Job{},
		WorkflowRunID: wr.ID,
		UserID:        admin.ID,
		Username:      admin.Username,
		ProjectKey:    wr.ProjectKey,
		JobID:         sdk.RandomString(10),
		Status:        sdk.StatusBuilding,
	}
	require.NoError(t, workflow_v2.InsertRunJob(context.TODO(), db, &wrj2))

	work2 := sdk.V2Worker{
		Status:       sdk.StatusBuilding,
		JobRunID:     wrj2.ID,
		HatcheryName: hatch.Name,
		HatcheryID:   hatch.ID,
		Name:         sdk.RandomString(10),
		LastBeat:     time.Now(),
	}
	require.NoError(t, worker_v2.Insert(ctx, db, &work2))

	workers, err := worker_v2.LoadWorkerByStatus(ctx, db, sdk.StatusDisabled)
	require.NoError(t, err)
	require.Equal(t, 1, len(workers))
	require.Equal(t, work.ID, workers[0].ID)

	require.NoError(t, worker_v2.DeleteDisabledWorker(ctx, cache, db.DbMap, workers[0].ID, workers[0].Name))

	_, err = worker_v2.LoadByID(ctx, db, workers[0].ID)
	require.True(t, sdk.ErrorIs(err, sdk.ErrNotFound))
}

func TestDisabledDeadWorkers(t *testing.T) {
	ctx := context.TODO()
	db, cache := test.SetupPG(t, bootstrap.InitiliazeDB)

	db.Exec("DELETE FROM v2_worker")
	db.Exec("DELETE FROM v2_workflow_run_job")

	admin, _ := assets.InsertAdminUser(t, db)
	proj := assets.InsertTestProject(t, db, cache, sdk.RandomString(10), sdk.RandomString(10))
	vcsServer := assets.InsertTestVCSProject(t, db, proj.ID, "github", "github")
	repo := assets.InsertTestProjectRepository(t, db, proj.Key, vcsServer.ID, sdk.RandomString(10))
	wr := sdk.V2WorkflowRun{
		ProjectKey:   proj.Key,
		VCSServerID:  vcsServer.ID,
		VCSServer:    vcsServer.Name,
		RepositoryID: repo.ID,
		Repository:   repo.Name,
		WorkflowName: sdk.RandomString(10),
		WorkflowSha:  "123",
		WorkflowRef:  "master",
		RunAttempt:   0,
		RunNumber:    1,
		Started:      time.Now(),
		LastModified: time.Now(),
		Status:       sdk.StatusBuilding,
		UserID:       admin.ID,
		Username:     admin.Username,
		RunEvent:     sdk.V2WorkflowRunEvent{},
		WorkflowData: sdk.V2WorkflowRunData{Workflow: sdk.V2Workflow{
			Jobs: map[string]sdk.V2Job{
				"job1": {},
				"job2": {
					Needs: []string{"job1"},
				},
			},
		}},
	}
	require.NoError(t, workflow_v2.InsertRun(context.Background(), db, &wr))

	hatch := sdk.Hatchery{
		ModelType: "docker",
		Name:      sdk.RandomString(10),
	}
	require.NoError(t, hatchery.Insert(ctx, db, &hatch))

	wrj := sdk.V2WorkflowRunJob{
		Job:           sdk.V2Job{},
		WorkflowRunID: wr.ID,
		UserID:        admin.ID,
		Username:      admin.Username,
		ProjectKey:    wr.ProjectKey,
		JobID:         sdk.RandomString(10),
		Status:        sdk.StatusBuilding,
	}
	require.NoError(t, workflow_v2.InsertRunJob(context.TODO(), db, &wrj))

	work := sdk.V2Worker{
		Status:       sdk.StatusBuilding,
		JobRunID:     wrj.ID,
		HatcheryName: hatch.Name,
		HatcheryID:   hatch.ID,
		Name:         sdk.RandomString(10),
		LastBeat:     time.Now().Add(-20 * time.Minute),
	}
	require.NoError(t, worker_v2.Insert(ctx, db, &work))

	wrj2 := sdk.V2WorkflowRunJob{
		Job:           sdk.V2Job{},
		WorkflowRunID: wr.ID,
		UserID:        admin.ID,
		Username:      admin.Username,
		ProjectKey:    wr.ProjectKey,
		JobID:         sdk.RandomString(10),
		Status:        sdk.StatusBuilding,
	}
	require.NoError(t, workflow_v2.InsertRunJob(context.TODO(), db, &wrj2))

	work2 := sdk.V2Worker{
		Status:       sdk.StatusBuilding,
		JobRunID:     wrj2.ID,
		HatcheryName: hatch.Name,
		HatcheryID:   hatch.ID,
		Name:         sdk.RandomString(10),
		LastBeat:     time.Now(),
	}
	require.NoError(t, worker_v2.Insert(ctx, db, &work2))

	workers, err := worker_v2.LoadDeadWorkers(ctx, db, 300.0, []string{sdk.StatusWaiting, sdk.StatusBuilding})
	require.NoError(t, err)
	require.Equal(t, 1, len(workers))
	require.Equal(t, work.ID, workers[0].ID)

	require.NoError(t, worker_v2.DisableDeadWorker(ctx, cache, db.DbMap, workers[0].ID, workers[0].Name))

	wrDB, err := worker_v2.LoadByID(ctx, db, workers[0].ID)
	require.NoError(t, err)
	require.Equal(t, sdk.StatusDisabled, wrDB.Status)
}
