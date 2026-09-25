package api

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/lib/pq"
	"github.com/rockbears/log"
	"go.opencensus.io/tag"

	"github.com/ovh/cds/engine/api/services"
	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/telemetry"
)

// A job that ran on a worker and produced no log item at all is the one failure of the log pipeline
// that no counter of that pipeline can report: the CDN counts what it received, and a job whose
// lines never reached it is, to the CDN, a job that does not exist. Only the API holds the list of
// jobs that ran, so only the API can say how many of them are missing from the CDN.
//
// What this measures, and what it does not:
//   - the denominator is the jobs that actually ran on a worker and are terminated. A job that was
//     never scheduled, was cancelled before starting or was skipped has no logs by definition and
//     is not counted as a loss.
//   - a job is looked at twice, after two different delays. At the short delay a missing item may
//     only be late, because the CDN creates the item when it dequeues the lines and the dequeue can
//     fall behind. At the long delay it is a loss. The distance between the two series is the
//     measure of the lateness.
//   - partial loss is measured apart, as a second series that never overlaps the first: a job with
//     some log items but fewer step log items than steps that ran. The intake counters of the CDN
//     cannot stand in for it, because lines lost with a connection the worker never saw die do not
//     reach the intake at all. A step that ran has exactly one log item, created from its first
//     line, and the service log items are left out of the count so that they cannot make up for the
//     steps that lost theirs. A step whose lines were only partly lost still has its item, so that
//     loss stays invisible here.
//   - partial loss is measured for v2 only. A terminated v1 job no longer has a row, and the only
//     trace of its steps is the copy kept in the stages of its node run, which has not been shown to
//     match the step log items one for one. No number is better than a wrong one: the v1 series is
//     left out.
//   - a job stopped in the middle of a step may count as partial when the step it was stopped in had
//     not written its first line yet. That step started, and nothing tells it apart from one whose
//     lines were lost.
const (
	// logCoverageInterval paces the check. The measurement is a window of the past, not a live
	// value, so it does not have to be refreshed faster than the window moves.
	logCoverageInterval = 5 * time.Minute
	// logCoverageWindow is how much of the past each pass looks at. Wide enough that a quiet
	// instance still has jobs to measure, narrow enough that one bad minute stays visible.
	logCoverageWindow = 30 * time.Minute
	// logCoverageBatch is how many job identifiers go into one call to the CDN. It matches what its
	// endpoint accepts.
	logCoverageBatch = 500
	// logCoverageMaxJobs bounds one pass per run version. Past it the pass measures a sample, which
	// keeps the ratio honest, and says so.
	logCoverageMaxJobs = 2000
	// logCoverageQueryTimeout caps the listing of a window. A listing that cannot complete within it
	// is worth losing rather than holding a connection for the next tick to pile onto.
	logCoverageQueryTimeout = 30 * time.Second
	// logCoveragePassTimeout caps a whole pass, the call to the CDN included. Nothing on that path
	// has a deadline of its own, and a CDN that answers no more would otherwise stall the ticker
	// for as long as it stays that way.
	logCoveragePassTimeout = 2 * time.Minute
	// logCoverageListLimit caps what the admin route returns. The metrics give the count; this gives
	// enough job identifiers to go and look at them one by one.
	logCoverageListLimit = 200
	// logCoverageCacheKey holds the counts one instance read for all of them.
	logCoverageCacheKey = "api:metrics:log-coverage"

	logCoverageRunVersionV1 = "v1"
	logCoverageRunVersionV2 = "v2"
)

// logCoverageGrace is one delay after which the jobs of a window are looked at. Two of them are
// used, and the difference between what they report is what separates a late item from a lost one.
type logCoverageGrace struct {
	name  string
	delay time.Duration
}

var logCoverageGraces = []logCoverageGrace{
	{name: "10min", delay: 10 * time.Minute},
	{name: "60min", delay: time.Hour},
}

// logCoverageResult is what one pass over one window found.
type logCoverageResult struct {
	checked map[string]int64
	missing map[string]int64
	// partial only has a key for the run versions it is measured for.
	partial     map[string]int64
	jobs        []sdk.LogCoverageJob
	partialJobs []sdk.LogCoverageJob
	truncated   bool
}

// computeLogCoverageMetrics refreshes the coverage gauges.
func (api *API) computeLogCoverageMetrics(ctx context.Context) {
	if !api.Config.LogCoverage.Enabled {
		return
	}
	api.GoRoutines.RunWithRestart(ctx, "api.computeLogCoverageMetrics", func(ctx context.Context) {
		tick := time.NewTicker(logCoverageInterval).C
		for {
			select {
			case <-ctx.Done():
				if ctx.Err() != nil {
					log.Error(ctx, "Exiting api.computeLogCoverageMetrics: %v", ctx.Err())
					return
				}
			case <-tick:
				api.refreshLogCoverageMetrics(ctx)
			}
		}
	})
}

// refreshLogCoverageMetrics records how many terminated jobs the CDN holds no log item for.
//
// The counts describe the two databases, so they are the same read from any instance, and reading
// them costs a window of the run tables plus a call to the CDN: one instance reads them and shares
// them, the others record what it read. Every instance records them rather than only the one that
// read them, because a measure only exists where it was recorded, and these views carry no tag that
// tells the instances apart: instances disagreeing would collapse into one series jumping between
// their values.
func (api *API) refreshLogCoverageMetrics(ctx context.Context) {
	// Held for the interval and not released: the instance that reads them is whichever one ticks
	// first once the last read has aged out.
	locked, err := api.Cache.Lock(cache.Key(logCoverageCacheKey, "lock"), logCoverageInterval, 0, 1)
	if err != nil {
		log.Warn(ctx, "metrics> unable to take the log coverage lock: %v", err)
	}

	if !locked {
		var shared map[string]int64
		found, err := api.Cache.Get(logCoverageCacheKey, &shared)
		if err != nil {
			log.Warn(ctx, "metrics> unable to read the shared log coverage counts: %v", err)
		}
		if found {
			api.recordLogCoverage(ctx, shared)
			return
		}
		// Nothing has been shared yet, which is the first pass of a cluster starting: reading it
		// here costs a duplicated pass, reporting nothing costs the metrics until the next tick.
	}

	now := time.Now()
	shared := make(map[string]int64, 2*len(logCoverageGraces)*2)
	for _, g := range logCoverageGraces {
		until := now.Add(-g.delay)
		since := until.Add(-logCoverageWindow)

		ctxPass, cancel := context.WithTimeout(ctx, logCoveragePassTimeout)
		res, err := api.collectLogCoverage(ctxPass, since, until, logCoverageMaxJobs)
		cancel()
		if err != nil {
			// Reporting nothing leaves the last counts in place. Reporting the zero of a pass that
			// did not happen reads as a window where nothing ran and nothing was lost.
			log.Warn(ctx, "metrics> unable to collect the log coverage of the %s window: %v", g.name, err)
			continue
		}
		if res.truncated {
			log.Warn(ctx, "metrics> the log coverage of the %s window hit the %d jobs cap: the ratio is a sample", g.name, logCoverageMaxJobs)
		}

		for _, v := range []string{logCoverageRunVersionV1, logCoverageRunVersionV2} {
			shared[logCoverageSharedKey(g.name, v, "checked")] = res.checked[v]
			shared[logCoverageSharedKey(g.name, v, "missing")] = res.missing[v]
			if partial, ok := res.partial[v]; ok {
				shared[logCoverageSharedKey(g.name, v, "partial")] = partial
			}
		}
	}

	api.recordLogCoverage(ctx, shared)

	if len(shared) == 0 {
		// Every window failed. Sharing that would replace counts the other instances can still
		// record with nothing at all.
		return
	}

	// Kept longer than the interval so that an instance ticking while no read is in progress still
	// finds them.
	if err := api.Cache.SetWithDuration(logCoverageCacheKey, shared, 3*logCoverageInterval); err != nil {
		log.Warn(ctx, "metrics> unable to share the log coverage counts: %v", err)
	}
}

func logCoverageSharedKey(timerange, runVersion, what string) string {
	return fmt.Sprintf("%s/%s/%s", timerange, runVersion, what)
}

// recordLogCoverage records what a pass found, or what the instance that ran it shared. A pass that
// failed leaves its keys out of the map and nothing is recorded for it.
func (api *API) recordLogCoverage(ctx context.Context, counts map[string]int64) {
	for _, g := range logCoverageGraces {
		for _, v := range []string{logCoverageRunVersionV1, logCoverageRunVersionV2} {
			checked, ok := counts[logCoverageSharedKey(g.name, v, "checked")]
			if !ok {
				continue
			}
			missing := counts[logCoverageSharedKey(g.name, v, "missing")]

			ctxTagged, err := tag.New(ctx, tag.Upsert(tagRange, g.name), tag.Upsert(tagRunVersion, v))
			if err != nil {
				log.Warn(ctx, "metrics> unable to tag the log coverage of %s/%s: %v", g.name, v, err)
				continue
			}
			telemetry.Record(ctxTagged, api.Metrics.logCoverageJobsChecked, checked)
			telemetry.Record(ctxTagged, api.Metrics.logCoverageJobsWithoutItem, missing)
			if partial, ok := counts[logCoverageSharedKey(g.name, v, "partial")]; ok {
				telemetry.Record(ctxTagged, api.Metrics.logCoverageJobsPartial, partial)
			}
		}
	}
}

// collectLogCoverage lists the jobs that ran on a worker and terminated inside the window, then
// asks the CDN how many log items it holds for each of them.
func (api *API) collectLogCoverage(ctx context.Context, since, until time.Time, limit int) (*logCoverageResult, error) {
	jobs, truncated, err := api.terminatedJobsInWindow(ctx, since, until, limit)
	if err != nil {
		return nil, err
	}

	var counts, stepCounts map[string]int64
	if len(jobs) > 0 {
		counts, stepCounts, err = api.countCDNLogItemsByJob(ctx, jobs)
		if err != nil {
			return nil, err
		}
	}

	return tallyLogCoverage(jobs, truncated, counts, stepCounts), nil
}

// tallyLogCoverage sorts the jobs of a window by what the CDN holds for them. A job with no log item
// at all is missing. A job with some, but fewer step log items than steps that ran, is partial. The
// two never overlap: a missing job is not also partial.
//
// A job absent from stepCounts is one the CDN could not count the step log items of, which is what
// a CDN that predates that count answers. It is never partial: taking the absence for a zero would
// report every job of the window as partial for as long as a deploy lasts.
func tallyLogCoverage(jobs []sdk.LogCoverageJob, truncated bool, counts, stepCounts map[string]int64) *logCoverageResult {
	res := &logCoverageResult{
		checked:   map[string]int64{logCoverageRunVersionV1: 0, logCoverageRunVersionV2: 0},
		missing:   map[string]int64{logCoverageRunVersionV1: 0, logCoverageRunVersionV2: 0},
		partial:   map[string]int64{logCoverageRunVersionV2: 0},
		truncated: truncated,
	}
	for _, j := range jobs {
		res.checked[j.RunVersion]++

		if counts[j.JobID] == 0 {
			res.missing[j.RunVersion]++
			res.jobs = append(res.jobs, j)
			continue
		}

		// The step count is only read for v2, so a v1 job never gets here.
		stepItems, known := stepCounts[j.JobID]
		if !known || j.StepCount == 0 || stepItems >= j.StepCount {
			continue
		}
		j.StepLogItemCount = stepItems
		res.partial[j.RunVersion]++
		res.partialJobs = append(res.partialJobs, j)
	}
	return res
}

// countCDNLogItemsByJob asks the CDN, by batches, how many log items it holds per job, in total and
// for the steps alone. The second map only has the jobs of the batches the CDN counted the step log
// items of.
func (api *API) countCDNLogItemsByJob(ctx context.Context, jobs []sdk.LogCoverageJob) (map[string]int64, map[string]int64, error) {
	srvs, err := services.LoadAllByType(ctx, api.mustDB(), sdk.TypeCDN)
	if err != nil {
		return nil, nil, err
	}
	if len(srvs) == 0 {
		return nil, nil, sdk.NewErrorFrom(sdk.ErrNotFound, "no cdn service registered")
	}

	counts := make(map[string]int64, len(jobs))
	stepCounts := make(map[string]int64, len(jobs))
	for start := 0; start < len(jobs); start += logCoverageBatch {
		end := start + logCoverageBatch
		if end > len(jobs) {
			end = len(jobs)
		}

		req := sdk.CDNJobLogCoverageRequest{JobIDs: make([]string, 0, end-start)}
		for _, j := range jobs[start:end] {
			req.JobIDs = append(req.JobIDs, j.JobID)
		}

		var resp sdk.CDNJobLogCoverageResponse
		if _, _, err := services.NewClient(srvs).DoJSONRequest(ctx, http.MethodPost, "/admin/items/log-coverage", req, &resp); err != nil {
			return nil, nil, err
		}
		for id, nb := range resp.LogItemCountByJobID {
			counts[id] = nb
		}
		// Absent from the answer of a CDN that predates the step count. The batches are not all
		// answered by the same instance, so this is decided batch by batch.
		if resp.StepLogItemCountByJobID != nil {
			for _, id := range req.JobIDs {
				stepCounts[id] = resp.StepLogItemCountByJobID[id]
			}
		}
	}
	return counts, stepCounts, nil
}

// terminatedJobsInWindow lists the jobs of both run versions that ran on a worker and terminated
// inside the window. A job with no worker name never reached one, so it has no logs to lose.
func (api *API) terminatedJobsInWindow(ctx context.Context, since, until time.Time, limit int) ([]sdk.LogCoverageJob, bool, error) {
	ctxQuery, cancel := context.WithTimeout(ctx, logCoverageQueryTimeout)
	defer cancel()
	db := api.mustDBWithCtx(ctxQuery)

	var jobs []sdk.LogCoverageJob
	var truncated bool

	type v2Row struct {
		ID           string    `db:"id"`
		Ended        time.Time `db:"ended"`
		ProjectKey   string    `db:"project_key"`
		WorkflowName string    `db:"workflow_name"`
		WorkerName   string    `db:"worker_name"`
		StepCount    int64     `db:"step_count"`
	}
	var v2Rows []v2Row
	// The steps that ran are the keys of steps_status, a json object held as text, counted here
	// rather than read back. The column is NULL on a job that never reached a step, and the text
	// "null" when the worker sent an empty map: both are zero, not an error.
	queryV2 := `
		SELECT id, ended, coalesce(project_key, '') as project_key, coalesce(workflow_name, '') as workflow_name, worker_name,
			case when jsonb_typeof(nullif(steps_status, '')::jsonb) = 'object'
				then (SELECT count(*) FROM jsonb_object_keys(steps_status::jsonb))
				else 0 end as step_count
		FROM v2_workflow_run_job
		WHERE ended >= $1 AND ended < $2
		AND status = ANY($3)
		AND coalesce(worker_name, '') <> ''
		ORDER BY ended
		LIMIT $4`
	if _, err := db.Select(&v2Rows, queryV2, since, until, pq.StringArray([]string{
		string(sdk.V2WorkflowRunJobStatusSuccess),
		string(sdk.V2WorkflowRunJobStatusFail),
		string(sdk.V2WorkflowRunJobStatusStopped),
	}), limit); err != nil {
		return nil, false, sdk.WithStack(err)
	}
	if len(v2Rows) >= limit {
		truncated = true
	}
	for _, r := range v2Rows {
		jobs = append(jobs, sdk.LogCoverageJob{
			JobID:        r.ID,
			RunVersion:   logCoverageRunVersionV2,
			Ended:        r.Ended,
			ProjectKey:   r.ProjectKey,
			WorkflowName: r.WorkflowName,
			WorkerName:   r.WorkerName,
			StepCount:    r.StepCount,
		})
	}

	// A terminated v1 job no longer has a row: WorkflowRunJobDeletion removes it once its node run
	// is over. What ran only survives in the stages of the node run, so the window is anchored on
	// the end of the node run, which is at or after the end of its jobs. The grace it sees is
	// therefore at least the one asked for.
	type v1Row struct {
		JobID      int64     `db:"job_id"`
		Done       time.Time `db:"done"`
		WorkerName string    `db:"worker_name"`
	}
	var v1Rows []v1Row
	queryV1 := `
		SELECT (job->>'id')::bigint as job_id, nr.done as done, coalesce(job->>'worker_name', '') as worker_name
		FROM workflow_node_run nr
		CROSS JOIN LATERAL jsonb_array_elements(
			case when jsonb_typeof(nr.stages) = 'array' then nr.stages else '[]'::jsonb end) AS stage
		CROSS JOIN LATERAL jsonb_array_elements(
			case when jsonb_typeof(stage->'run_jobs') = 'array' then stage->'run_jobs' else '[]'::jsonb end) AS job
		WHERE nr.done >= $1 AND nr.done < $2
		AND job->>'status' = ANY($3)
		AND coalesce(job->>'worker_name', '') <> ''
		ORDER BY nr.done
		LIMIT $4`
	if _, err := db.Select(&v1Rows, queryV1, since, until, pq.StringArray([]string{
		sdk.StatusSuccess,
		sdk.StatusFail,
		sdk.StatusStopped,
	}), limit); err != nil {
		return nil, false, sdk.WithStack(err)
	}
	if len(v1Rows) >= limit {
		truncated = true
	}
	for _, r := range v1Rows {
		jobs = append(jobs, sdk.LogCoverageJob{
			JobID:      strconv.FormatInt(r.JobID, 10),
			RunVersion: logCoverageRunVersionV1,
			Ended:      r.Done,
			WorkerName: r.WorkerName,
		})
	}

	return jobs, truncated, nil
}

// getAdminLogCoverageHandler answers with the jobs of a window the CDN holds no log item for, and
// the ones it holds fewer step log items for than steps ran. It is what makes the coverage metrics
// actionable: each identifier it returns can be handed to the debug route of the CDN to find out
// where the lines of that job stopped.
func (api *API) getAdminLogCoverageHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		until := time.Now().Add(-logCoverageGraces[0].delay)
		since := until.Add(-logCoverageWindow)

		if s := FormString(r, "since"); s != "" {
			t, err := time.Parse(time.RFC3339, s)
			if err != nil {
				return sdk.NewErrorFrom(sdk.ErrWrongRequest, "invalid since, expected RFC3339: %v", err)
			}
			since = t
		}
		if s := FormString(r, "until"); s != "" {
			t, err := time.Parse(time.RFC3339, s)
			if err != nil {
				return sdk.NewErrorFrom(sdk.ErrWrongRequest, "invalid until, expected RFC3339: %v", err)
			}
			until = t
		}
		if !until.After(since) {
			return sdk.NewErrorFrom(sdk.ErrWrongRequest, "until must be after since")
		}

		res, err := api.collectLogCoverage(ctx, since, until, logCoverageMaxJobs)
		if err != nil {
			return err
		}

		report := sdk.LogCoverageReport{
			Since:               since,
			Until:               until,
			JobsChecked:         res.checked[logCoverageRunVersionV1] + res.checked[logCoverageRunVersionV2],
			JobsWithoutItem:     res.jobs,
			JobsWithPartialLogs: res.partialJobs,
			Truncated:           res.truncated,
		}
		if len(report.JobsWithoutItem) > logCoverageListLimit {
			report.JobsWithoutItem = report.JobsWithoutItem[:logCoverageListLimit]
			report.Truncated = true
		}
		if len(report.JobsWithPartialLogs) > logCoverageListLimit {
			report.JobsWithPartialLogs = report.JobsWithPartialLogs[:logCoverageListLimit]
			report.Truncated = true
		}

		return service.WriteJSON(w, report, http.StatusOK)
	}
}
