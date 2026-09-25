package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"

	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/engine/api/user"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/engine/api/workflow_v2"
	"github.com/ovh/cds/sdk"
)

// TestRecordLogCoverage pins the rule the whole measurement rests on: a window whose pass failed
// records nothing at all. Recording the zero of a pass that did not happen would read as a window
// where everything ran and nothing was lost, which is the one wrong answer this metric can give.
// It also pins that every series carries both the delay it was measured after and the run version,
// since neither number means anything without them.
func TestRecordLogCoverage(t *testing.T) {
	tagRange, _ = tag.NewKey("range")
	tagRunVersion, _ = tag.NewKey("run_version")

	api := &API{}
	api.Metrics.logCoverageJobsChecked = stats.Int64("test/log_coverage_jobs_checked", "", stats.UnitDimensionless)
	api.Metrics.logCoverageJobsWithoutItem = stats.Int64("test/log_coverage_jobs_without_item", "", stats.UnitDimensionless)
	api.Metrics.logCoverageJobsPartial = stats.Int64("test/log_coverage_jobs_partial", "", stats.UnitDimensionless)

	checked := &view.View{
		Name:        api.Metrics.logCoverageJobsChecked.Name(),
		Measure:     api.Metrics.logCoverageJobsChecked,
		Aggregation: view.LastValue(),
		TagKeys:     []tag.Key{tagRange, tagRunVersion},
	}
	missing := &view.View{
		Name:        api.Metrics.logCoverageJobsWithoutItem.Name(),
		Measure:     api.Metrics.logCoverageJobsWithoutItem,
		Aggregation: view.LastValue(),
		TagKeys:     []tag.Key{tagRange, tagRunVersion},
	}
	partial := &view.View{
		Name:        api.Metrics.logCoverageJobsPartial.Name(),
		Measure:     api.Metrics.logCoverageJobsPartial,
		Aggregation: view.LastValue(),
		TagKeys:     []tag.Key{tagRange, tagRunVersion},
	}
	require.NoError(t, view.Register(checked, missing, partial))
	t.Cleanup(func() { view.Unregister(checked, missing, partial) })

	// Only the 60min window of v2 answered. The 10min window and v1 are left out, as a failed pass
	// leaves them out.
	api.recordLogCoverage(context.TODO(), map[string]int64{
		logCoverageSharedKey("60min", logCoverageRunVersionV2, "checked"): 120,
		logCoverageSharedKey("60min", logCoverageRunVersionV2, "missing"): 3,
		logCoverageSharedKey("60min", logCoverageRunVersionV2, "partial"): 2,
	})

	require.Equal(t, map[string]int64{"60min/v2": 120}, lastValueByRangeAndVersion(t, checked.Name))
	require.Equal(t, map[string]int64{"60min/v2": 3}, lastValueByRangeAndVersion(t, missing.Name))
	require.Equal(t, map[string]int64{"60min/v2": 2}, lastValueByRangeAndVersion(t, partial.Name))

	// A job count of zero is a real answer and has to be recorded, unlike a missing one. Partial loss
	// is not measured for v1, and a v1 series at zero would claim it was and found none.
	api.recordLogCoverage(context.TODO(), map[string]int64{
		logCoverageSharedKey("10min", logCoverageRunVersionV1, "checked"): 0,
		logCoverageSharedKey("10min", logCoverageRunVersionV1, "missing"): 0,
	})
	require.Equal(t, map[string]int64{"60min/v2": 120, "10min/v1": 0}, lastValueByRangeAndVersion(t, checked.Name))
	require.Equal(t, map[string]int64{"60min/v2": 2}, lastValueByRangeAndVersion(t, partial.Name))
}

// TestTallyLogCoverage pins what a partial loss is. It is the job the first series cannot see: some
// of its steps have a log item, so it is not missing, and the others do not. It must not be confused
// with a missing job, or the two series would count the same loss twice. Service log items must not
// hide it, or a job with services whose step logs are all gone would look complete. And a step count
// or a step log item count that is not known must never be read as a loss: a job that never reached
// a step, a v1 job, or a CDN that predates the step count would otherwise report partial loss on
// every job of the window.
func TestTallyLogCoverage(t *testing.T) {
	v2Job := func(id string, steps int64) sdk.LogCoverageJob {
		return sdk.LogCoverageJob{JobID: id, RunVersion: logCoverageRunVersionV2, StepCount: steps}
	}
	jobs := []sdk.LogCoverageJob{
		v2Job("complete", 3),
		v2Job("lost-half", 8),                              // (a) 4 steps of 8 have their item
		v2Job("no-item", 8),                                // (b) nothing at all: missing, not partial
		v2Job("services-only", 3),                          // (c) only the items of its services
		v2Job("no-step-count", 0),                          // (d) never reached a step
		v2Job("old-cdn", 5),                                // (d) answered by a CDN that does not count the steps
		v2Job("more-items", 2),                             // more items than steps is not a loss
		{JobID: "v1", RunVersion: logCoverageRunVersionV1}, // (d) v1 carries no step count
	}
	counts := map[string]int64{
		"complete":      3,
		"lost-half":     4,
		"services-only": 2,
		"no-step-count": 1,
		"old-cdn":       1,
		"more-items":    3,
		"v1":            1,
	}
	stepCounts := map[string]int64{
		"complete":      3,
		"lost-half":     4,
		"no-item":       0,
		"services-only": 0,
		"no-step-count": 1,
		"more-items":    3,
		"v1":            0,
	}

	res := tallyLogCoverage(jobs, false, counts, stepCounts)

	require.Equal(t, map[string]int64{logCoverageRunVersionV1: 1, logCoverageRunVersionV2: 7}, res.checked)
	require.Equal(t, map[string]int64{logCoverageRunVersionV1: 0, logCoverageRunVersionV2: 1}, res.missing)
	require.Equal(t, map[string]int64{logCoverageRunVersionV2: 2}, res.partial, "no v1 key: partial loss is not measured for v1")

	require.Len(t, res.jobs, 1)
	require.Equal(t, "no-item", res.jobs[0].JobID, "a job with no item at all is missing, and only missing")

	partialIDs := make([]string, 0, len(res.partialJobs))
	for _, j := range res.partialJobs {
		partialIDs = append(partialIDs, j.JobID)
	}
	require.Equal(t, []string{"lost-half", "services-only"}, partialIDs)
	require.Equal(t, int64(4), res.partialJobs[0].StepLogItemCount, "the listing says how many steps kept their logs")
	require.Equal(t, int64(0), res.partialJobs[1].StepLogItemCount)

	// A whole window answered by a CDN that predates the step count reports no partial loss at all.
	old := tallyLogCoverage(jobs, false, counts, map[string]int64{})
	require.Equal(t, int64(0), old.partial[logCoverageRunVersionV2])
	require.Empty(t, old.partialJobs)
	require.Equal(t, res.missing, old.missing, "the first series does not depend on the step count")

	// What tells a CDN that predates the step count from one that counted nothing is whether the
	// field is there at all, zero entries included.
	var fromOld, fromNew sdk.CDNJobLogCoverageResponse
	require.NoError(t, json.Unmarshal([]byte(`{"log_item_count_by_job_id":{"a":1}}`), &fromOld))
	require.NoError(t, json.Unmarshal([]byte(`{"log_item_count_by_job_id":{},"step_log_item_count_by_job_id":{}}`), &fromNew))
	require.Nil(t, fromOld.StepLogItemCountByJobID)
	require.NotNil(t, fromNew.StepLogItemCountByJobID)
}

func lastValueByRangeAndVersion(t *testing.T, viewName string) map[string]int64 {
	rows, err := view.RetrieveData(viewName)
	require.NoError(t, err)

	res := make(map[string]int64, len(rows))
	for _, r := range rows {
		data, ok := r.Data.(*view.LastValueData)
		require.True(t, ok)

		var timerange, runVersion string
		for _, tg := range r.Tags {
			switch tg.Key.Name() {
			case "range":
				timerange = tg.Value
			case "run_version":
				runVersion = tg.Value
			}
		}
		res[fmt.Sprintf("%s/%s", timerange, runVersion)] = int64(data.Value)
	}
	return res
}

// TestTerminatedJobsInWindow pins the denominator of the coverage measurement, which is a
// definition and not a detail: a job counts only if it ran on a worker and terminated inside the
// window. A job that never reached a worker has no logs to lose, and counting it would invent a
// loss that never happened.
func TestTerminatedJobsInWindow(t *testing.T) {
	ctx := context.TODO()
	api, db, _ := newTestAPI(t)

	db.Exec("DELETE FROM v2_workflow_run_job")

	admin, _ := assets.InsertAdminUser(t, db)
	admin, _ = user.LoadByID(ctx, db, admin.ID, user.LoadOptions.WithContacts)

	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))
	vcsServer := assets.InsertTestVCSProject(t, db, proj.ID, "github", "github")
	repo := assets.InsertTestProjectRepository(t, db, proj.Key, vcsServer.ID, sdk.RandomString(10))

	wr := sdk.V2WorkflowRun{
		ProjectKey:         proj.Key,
		VCSServerID:        vcsServer.ID,
		VCSServer:          vcsServer.Name,
		RepositoryID:       repo.ID,
		Repository:         repo.Name,
		WorkflowName:       sdk.RandomString(10),
		WorkflowSha:        "123",
		WorkflowRef:        "master",
		RunAttempt:         0,
		RunNumber:          1,
		Started:            time.Now(),
		LastModified:       time.Now(),
		Status:             sdk.StatusBuilding,
		DeprecatedUserID:   admin.ID,
		DeprecatedUsername: admin.Username,
		Initiator:          &sdk.V2Initiator{UserID: admin.ID, User: admin.Initiator()},
		RunEvent:           sdk.V2WorkflowRunEvent{},
		WorkflowData:       sdk.V2WorkflowRunData{Workflow: sdk.V2Workflow{Jobs: map[string]sdk.V2Job{"job1": {}}}},
	}
	require.NoError(t, workflow_v2.InsertRun(ctx, db, &wr))

	now := time.Now()
	since := now.Add(-time.Hour)
	until := now.Add(-10 * time.Minute)
	inWindow := now.Add(-30 * time.Minute)

	insertJob := func(status sdk.V2WorkflowRunJobStatus, workerName string, ended time.Time) string {
		wrj := sdk.V2WorkflowRunJob{
			Job:           sdk.V2Job{},
			WorkflowRunID: wr.ID,
			ProjectKey:    wr.ProjectKey,
			WorkflowName:  wr.WorkflowName,
			JobID:         sdk.RandomString(10),
			Status:        status,
			WorkerName:    workerName,
			Initiator:     sdk.V2Initiator{UserID: admin.ID, User: admin.Initiator()},
		}
		require.NoError(t, workflow_v2.InsertRunJob(ctx, db, &wrj))
		// The end date is written by the engine when the job terminates, not by the insert.
		_, err := db.Exec("UPDATE v2_workflow_run_job SET ended = $1 WHERE id = $2", ended, wrj.ID)
		require.NoError(t, err)
		return wrj.ID
	}

	kept := insertJob(sdk.V2WorkflowRunJobStatusSuccess, "worker-1", inWindow)
	insertJob(sdk.V2WorkflowRunJobStatusSuccess, "", inWindow)                        // never reached a worker
	insertJob(sdk.V2WorkflowRunJobStatusSkipped, "worker-2", inWindow)                // never ran
	insertJob(sdk.V2WorkflowRunJobStatusSuccess, "worker-3", now.Add(-5*time.Minute)) // too recent, still inside the grace
	insertJob(sdk.V2WorkflowRunJobStatusSuccess, "worker-4", now.Add(-2*time.Hour))   // older than the window
	failed := insertJob(sdk.V2WorkflowRunJobStatusFail, "worker-5", inWindow)         // a failed job still logs
	noStepStatus := insertJob(sdk.V2WorkflowRunJobStatusFail, "worker-6", inWindow)
	emptyStepStatus := insertJob(sdk.V2WorkflowRunJobStatusFail, "worker-7", inWindow)

	// The steps that ran are counted out of steps_status, which is text and not always a json object.
	// Whatever it holds, the listing must not fail: failing would silence the whole window.
	setStepsStatus := func(id string, value interface{}) {
		_, err := db.Exec("UPDATE v2_workflow_run_job SET steps_status = $1 WHERE id = $2", value, id)
		require.NoError(t, err)
	}
	setStepsStatus(kept, `{"step-0":{},"step-1":{},"Post-step-0":{}}`)
	setStepsStatus(failed, `null`) // what an empty map of steps is written as
	setStepsStatus(noStepStatus, nil)
	setStepsStatus(emptyStepStatus, "")

	jobs, truncated, err := api.terminatedJobsInWindow(ctx, since, until, logCoverageMaxJobs)
	require.NoError(t, err)
	require.False(t, truncated)

	found := make(map[string]sdk.LogCoverageJob, len(jobs))
	for _, j := range jobs {
		if j.RunVersion == logCoverageRunVersionV2 {
			found[j.JobID] = j
		}
	}
	require.Len(t, found, 4, "only the jobs that ran on a worker and ended inside the window are counted")
	require.Contains(t, found, kept)
	require.Contains(t, found, failed)
	require.Equal(t, "worker-1", found[kept].WorkerName)
	require.Equal(t, proj.Key, found[kept].ProjectKey)
	require.Equal(t, wr.WorkflowName, found[kept].WorkflowName)

	require.Equal(t, int64(3), found[kept].StepCount, "every step that ran counts, a post step included")
	for _, id := range []string{failed, noStepStatus, emptyStepStatus} {
		require.Equal(t, int64(0), found[id].StepCount, "a job with no step recorded has an unknown step count, never an error")
	}
}

// TestTerminatedJobsInWindowV1 pins the v1 half of the denominator. A terminated v1 job no longer
// has a row of its own, so it is read back out of the stages of its node run, and that document is
// not guaranteed to be an array of stages each holding an array of jobs: a node run that never
// reached a stage has none. The query has to walk past those without failing, because failing would
// silence the measurement for every job of the window, not just for that node run.
func TestTerminatedJobsInWindowV1(t *testing.T) {
	ctx := context.TODO()
	api, db, _ := newTestAPI(t)

	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))
	wk := sdk.Workflow{
		Name:         sdk.RandomString(10),
		ProjectKey:   proj.Key,
		ProjectID:    proj.ID,
		WorkflowData: sdk.WorkflowData{Node: sdk.Node{Name: "root"}},
	}
	require.NoError(t, workflow.Insert(ctx, db, api.Cache, *proj, &wk))

	wr := workflow.Run{WorkflowID: wk.ID, Workflow: wk, ProjectID: proj.ID}
	require.NoError(t, db.Insert(&wr))

	now := time.Now()
	since := now.Add(-time.Hour)
	until := now.Add(-10 * time.Minute)
	inWindow := now.Add(-30 * time.Minute)

	insertNodeRun := func(subNum int64, done time.Time, stages sql.NullString) {
		nr := workflow.NodeRun{
			WorkflowRunID:  wr.ID,
			WorkflowID:     sql.NullInt64{Int64: wk.ID, Valid: true},
			WorkflowNodeID: wk.WorkflowData.Node.ID,
			Number:         1,
			SubNumber:      subNum,
			Status:         sdk.StatusSuccess,
			Start:          done.Add(-time.Minute),
			Done:           done,
			LastModified:   done,
			Stages:         stages,
		}
		require.NoError(t, db.Insert(&nr))
	}

	jsonStages := func(s string) sql.NullString { return sql.NullString{String: s, Valid: true} }

	// The node runs of earlier tests stay in the database, so the job identifiers are derived from
	// this run and the assertions only look at them.
	base := wr.ID * 10
	ran, noWorker, skipped, tooOld := base+1, base+2, base+3, base+4

	insertNodeRun(1, inWindow, jsonStages(fmt.Sprintf(`[{"run_jobs":[
		{"id":%d,"status":"Success","worker_name":"w1"},
		{"id":%d,"status":"Success","worker_name":""},
		{"id":%d,"status":"Skipped","worker_name":"w3"}]}]`, ran, noWorker, skipped)))
	insertNodeRun(2, inWindow, sql.NullString{})                  // no stage at all
	insertNodeRun(3, inWindow, jsonStages(`[{"run_jobs":null}]`)) // a stage that ran no job
	insertNodeRun(4, now.Add(-2*time.Hour), jsonStages(fmt.Sprintf(`[{"run_jobs":[
		{"id":%d,"status":"Success","worker_name":"w4"}]}]`, tooOld))) // older than the window

	jobs, truncated, err := api.terminatedJobsInWindow(ctx, since, until, logCoverageMaxJobs)
	require.NoError(t, err)
	require.False(t, truncated)

	mine := make(map[string]sdk.LogCoverageJob)
	for _, j := range jobs {
		if j.RunVersion != logCoverageRunVersionV1 {
			continue
		}
		switch j.JobID {
		case strconv.FormatInt(ran, 10), strconv.FormatInt(noWorker, 10), strconv.FormatInt(skipped, 10), strconv.FormatInt(tooOld, 10):
			mine[j.JobID] = j
		}
	}
	require.Len(t, mine, 1, "only the job that ran on a worker and ended inside the window is counted")
	require.Contains(t, mine, strconv.FormatInt(ran, 10))
	require.Equal(t, "w1", mine[strconv.FormatInt(ran, 10)].WorkerName)
}
