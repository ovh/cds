package api

import (
	"context"
	"github.com/ovh/cds/engine/api/hatchery"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/engine/api/worker_v2"
	"github.com/ovh/cds/engine/api/workflow_v2"
	"github.com/ovh/cds/sdk"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestReEnqueueScheduledJobs(t *testing.T) {
	ctx := context.TODO()
	api, db, _ := newTestAPI(t)

	db.Exec("DELETE FROM v2_worker")
	db.Exec("DELETE FROM v2_workflow_run_job")

	admin, _ := assets.InsertAdminUser(t, db)
	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))
	vcsServer := assets.InsertTestVCSProject(t, db, proj.ID, "github", "github")
	repo := assets.InsertTestProjectRepository(t, db, vcsServer.ID, sdk.RandomString(10))
	wr := sdk.V2WorkflowRun{
		ProjectKey:   proj.Key,
		VCSServerID:  vcsServer.ID,
		RepositoryID: repo.ID,
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
		Event:        sdk.V2WorkflowRunEvent{},
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

	wrj := sdk.V2WorkflowRunJob{
		Job:           sdk.V2Job{},
		WorkflowRunID: wr.ID,
		Outputs:       sdk.JobResultOutput{},
		UserID:        admin.ID,
		Username:      admin.Username,
		ProjectKey:    wr.ProjectKey,
		Scheduled:     time.Now().Add(-20 * time.Minute),
		JobID:         sdk.RandomString(10),
		Status:        sdk.StatusScheduling,
	}
	require.NoError(t, workflow_v2.InsertRunJob(context.TODO(), db, &wrj))

	wrj2 := sdk.V2WorkflowRunJob{
		Job:           sdk.V2Job{},
		WorkflowRunID: wr.ID,
		Outputs:       sdk.JobResultOutput{},
		UserID:        admin.ID,
		Username:      admin.Username,
		ProjectKey:    wr.ProjectKey,
		Scheduled:     time.Now(),
		JobID:         sdk.RandomString(10),
		Status:        sdk.StatusScheduling,
	}
	require.NoError(t, workflow_v2.InsertRunJob(context.TODO(), db, &wrj2))

	jobs, err := workflow_v2.LoadOldScheduledRunJob(ctx, api.mustDB(), jobSchedulingTimeout)
	require.NoError(t, err)
	require.Equal(t, len(jobs), 1)
	require.Equal(t, wrj.ID, jobs[0].ID)

	require.NoError(t, reEnqueueScheduledJob(ctx, api.Cache, api.mustDB(), jobs[0].ID))

	rjDB, err := workflow_v2.LoadRunJobByID(ctx, db, jobs[0].ID)
	require.NoError(t, err)
	require.Equal(t, sdk.StatusWaiting, rjDB.Status)
}

func TestStopDeadJobs(t *testing.T) {
	ctx := context.TODO()
	api, db, _ := newTestAPI(t)

	db.Exec("DELETE FROM v2_worker")
	db.Exec("DELETE FROM v2_workflow_run_job")

	admin, _ := assets.InsertAdminUser(t, db)
	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))
	vcsServer := assets.InsertTestVCSProject(t, db, proj.ID, "github", "github")
	repo := assets.InsertTestProjectRepository(t, db, vcsServer.ID, sdk.RandomString(10))
	wr := sdk.V2WorkflowRun{
		ProjectKey:   proj.Key,
		VCSServerID:  vcsServer.ID,
		RepositoryID: repo.ID,
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
		Event:        sdk.V2WorkflowRunEvent{},
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
		Outputs:       sdk.JobResultOutput{},
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
	}
	require.NoError(t, worker_v2.Insert(ctx, db, &work))

	wrj2 := sdk.V2WorkflowRunJob{
		Job:           sdk.V2Job{},
		WorkflowRunID: wr.ID,
		Outputs:       sdk.JobResultOutput{},
		UserID:        admin.ID,
		Username:      admin.Username,
		ProjectKey:    wr.ProjectKey,
		JobID:         sdk.RandomString(10),
		Status:        sdk.StatusBuilding,
	}
	require.NoError(t, workflow_v2.InsertRunJob(context.TODO(), db, &wrj2))

	jobs, err := workflow_v2.LoadDeadJobs(ctx, db)
	require.NoError(t, err)
	require.Equal(t, 1, len(jobs))
	require.Equal(t, wrj2.ID, jobs[0].ID)

	require.NoError(t, stopDeadJob(ctx, api.Cache, db.DbMap, wrj2.ID))

	rjDB, err := workflow_v2.LoadRunJobByID(ctx, db, jobs[0].ID)
	require.NoError(t, err)
	require.Equal(t, sdk.StatusStopped, rjDB.Status)
}
