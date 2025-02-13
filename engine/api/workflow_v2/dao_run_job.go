package workflow_v2

import (
	"context"
	"time"

	"github.com/go-gorp/gorp"
	"github.com/lib/pq"
	"github.com/rockbears/log"
	"go.opencensus.io/trace"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/api/user"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/telemetry"
)

func getAllRunJobs(ctx context.Context, db gorp.SqlExecutor, query gorpmapping.Query) ([]sdk.V2WorkflowRunJob, error) {
	var dbWkfRunJobs []dbWorkflowRunJob
	if err := gorpmapping.GetAll(ctx, db, query, &dbWkfRunJobs); err != nil {
		return nil, err
	}
	jobRuns := make([]sdk.V2WorkflowRunJob, 0, len(dbWkfRunJobs))
	for _, rj := range dbWkfRunJobs {
		isValid, err := gorpmapping.CheckSignature(rj, rj.Signature)
		if err != nil {
			return nil, err
		}
		if !isValid {
			log.Error(ctx, "run job %s on run %s: data corrupted", rj.ID, rj.WorkflowRunID)
			continue
		}
		if rj.Initiator.UserID == "" {
			rj.Initiator.UserID = rj.DeprecatedUserID
		}
		if rj.Initiator.UserID != "" && rj.Initiator.User == nil {
			u, err := user.LoadByID(ctx, db, rj.Initiator.UserID, user.LoadOptions.WithContacts)
			if err != nil {
				return nil, err
			}
			rj.Initiator.User = u.Initiator()
		}
		jobRuns = append(jobRuns, rj.V2WorkflowRunJob)
	}
	return jobRuns, nil
}

func getRunJob(ctx context.Context, db gorp.SqlExecutor, query gorpmapping.Query) (*sdk.V2WorkflowRunJob, error) {
	var dbWkfRunJob dbWorkflowRunJob
	found, err := gorpmapping.Get(ctx, db, query, &dbWkfRunJob)
	if err != nil {
		return nil, sdk.WithStack(err)
	}
	if !found {
		return nil, sdk.WithStack(sdk.ErrNotFound)
	}
	isValid, err := gorpmapping.CheckSignature(dbWkfRunJob, dbWkfRunJob.Signature)
	if err != nil {
		return nil, err
	}
	if !isValid {
		log.Error(ctx, "run job %s on run %s: data corrupted", dbWkfRunJob.ID, dbWkfRunJob.WorkflowRunID)
		return nil, sdk.WithStack(sdk.ErrNotFound)
	}
	if dbWkfRunJob.Initiator.UserID == "" {
		dbWkfRunJob.Initiator.UserID = dbWkfRunJob.DeprecatedUserID
	}
	if dbWkfRunJob.Initiator.UserID != "" && dbWkfRunJob.Initiator.User == nil {
		u, err := user.LoadByID(ctx, db, dbWkfRunJob.Initiator.UserID, user.LoadOptions.WithContacts)
		if err != nil {
			return nil, err
		}
		dbWkfRunJob.Initiator.User = u.Initiator()
	}
	return &dbWkfRunJob.V2WorkflowRunJob, nil
}

func InsertRunJob(ctx context.Context, db gorpmapper.SqlExecutorWithTx, wrj *sdk.V2WorkflowRunJob) error {
	ctx, next := telemetry.Span(ctx, "workflow_v2.InsertRunJob", trace.StringAttribute(telemetry.TagJob, wrj.JobID))
	defer next()
	if wrj.ID == "" {
		wrj.ID = sdk.UUID()
	}
	if wrj.Queued.IsZero() {
		wrj.Queued = time.Now()
	}
	dbWkfRunJob := &dbWorkflowRunJob{V2WorkflowRunJob: *wrj}

	// Compat code
	dbWkfRunJob.DeprecatedUserID = dbWkfRunJob.Initiator.UserID
	dbWkfRunJob.DeprecatedAdminMFA = dbWkfRunJob.Initiator.IsAdminWithMFA
	dbWkfRunJob.DeprecatedUsername = dbWkfRunJob.Initiator.Username()

	if dbWkfRunJob.Initiator.UserID == "" && dbWkfRunJob.Initiator.VCSUsername == "" {
		return sdk.NewErrorFrom(sdk.ErrUnknownError, "V2WorkflowRunJob initiator should not be nil")
	}

	if err := gorpmapping.InsertAndSign(ctx, db, dbWkfRunJob); err != nil {
		return err
	}
	*wrj = dbWkfRunJob.V2WorkflowRunJob
	return nil
}

func UpdateJobRun(ctx context.Context, db gorpmapper.SqlExecutorWithTx, wrj *sdk.V2WorkflowRunJob) error {
	ctx, next := telemetry.Span(ctx, "workflow_v2.UpdateJobRun")
	defer next()
	dbWkfRunJob := &dbWorkflowRunJob{V2WorkflowRunJob: *wrj}
	dbWkfRunJob.DeprecatedUserID = dbWkfRunJob.Initiator.UserID
	dbWkfRunJob.DeprecatedAdminMFA = dbWkfRunJob.Initiator.IsAdminWithMFA
	dbWkfRunJob.DeprecatedUsername = dbWkfRunJob.Initiator.Username()
	if err := gorpmapping.UpdateAndSign(ctx, db, dbWkfRunJob); err != nil {
		return err
	}
	*wrj = dbWkfRunJob.V2WorkflowRunJob
	return nil
}

func LoadRunJobsByRunID(ctx context.Context, db gorp.SqlExecutor, runID string, runAttempt int64) ([]sdk.V2WorkflowRunJob, error) {
	ctx, next := telemetry.Span(ctx, "LoadRunJobsByRunID")
	defer next()
	query := gorpmapping.NewQuery(`
		SELECT *
		FROM v2_workflow_run_job
		WHERE workflow_run_id = $1 AND run_attempt = $2
	`).Args(runID, runAttempt)
	return getAllRunJobs(ctx, db, query)
}

func UnsafeLoadAllRunJobs(ctx context.Context, db gorp.SqlExecutor) ([]sdk.V2WorkflowRunJob, error) {
	query := "SELECT * from v2_workflow_run_job"
	var runJobs []sdk.V2WorkflowRunJob
	if _, err := db.Select(&runJobs, query); err != nil {
		return nil, sdk.WithStack(err)
	}
	return runJobs, nil
}

func LoadRunJobByID(ctx context.Context, db gorp.SqlExecutor, jobRunID string) (*sdk.V2WorkflowRunJob, error) {
	ctx, next := telemetry.Span(ctx, "workflow_v2.LoadRunJobByID")
	defer next()
	query := gorpmapping.NewQuery("SELECT * from v2_workflow_run_job WHERE id = $1").Args(jobRunID)
	return getRunJob(ctx, db, query)
}

func LoadRunJobByRunIDAndID(ctx context.Context, db gorp.SqlExecutor, wrID, jobRunID string) (*sdk.V2WorkflowRunJob, error) {
	ctx, next := telemetry.Span(ctx, "workflow_v2.LoadRunJobByRunIDAndID")
	defer next()
	query := gorpmapping.NewQuery("SELECT * from v2_workflow_run_job WHERE workflow_run_id = $1 AND id = $2").Args(wrID, jobRunID)
	return getRunJob(ctx, db, query)
}

func LoadRunJobsByName(ctx context.Context, db gorp.SqlExecutor, wrID string, jobName string, runAttempt int64) ([]sdk.V2WorkflowRunJob, error) {
	ctx, next := telemetry.Span(ctx, "workflow_v2.LoadRunJobsByName")
	defer next()
	query := gorpmapping.NewQuery("SELECT * from v2_workflow_run_job WHERE workflow_run_id = $1 AND job_id = $2 AND run_attempt = $3").Args(wrID, jobName, runAttempt)
	return getAllRunJobs(ctx, db, query)
}

func LoadQueuedRunJobByModelTypeAndRegionAndModelOSArch(ctx context.Context, db gorp.SqlExecutor, regionName string, modelType string, modelOSArch string) ([]sdk.V2WorkflowRunJob, error) {
	ctx, next := telemetry.Span(ctx, "workflow_v2.LoadQueuedRunJobByModelTypeAndRegion")
	defer next()
	query := gorpmapping.NewQuery("SELECT * from v2_workflow_run_job WHERE status = $1 AND model_type = $2 and region = $3 and model_osarch = $4 ORDER BY queued").
		Args(sdk.StatusWaiting, modelType, regionName, modelOSArch)
	return getAllRunJobs(ctx, db, query)
}

func LoadRunJobsByRunIDAndStatus(ctx context.Context, db gorp.SqlExecutor, runID string, status []string, runAttempt int64) ([]sdk.V2WorkflowRunJob, error) {
	ctx, next := telemetry.Span(ctx, "workflow_v2.LoadRunJobsByRunIDAndStatus")
	defer next()
	query := gorpmapping.NewQuery("SELECT * from v2_workflow_run_job WHERE workflow_run_id = $1 AND status = ANY($2) AND run_attempt = $3").Args(runID, pq.StringArray(status), runAttempt)
	return getAllRunJobs(ctx, db, query)
}

func LoadOldScheduledRunJob(ctx context.Context, db gorp.SqlExecutor, timeout int64) ([]sdk.V2WorkflowRunJob, error) {
	ctx, next := telemetry.Span(ctx, "workflow_v2.LoadOldScheduledRunJob")
	defer next()
	query := gorpmapping.NewQuery(`
    SELECT *
    FROM v2_workflow_run_job
    WHERE status = $1 AND now() - scheduled > $2 * INTERVAL '1' SECOND
    LIMIT 100
    `).Args(sdk.StatusScheduling, timeout)
	return getAllRunJobs(ctx, db, query)
}

func LoadOldWaitingRunJob(ctx context.Context, db gorp.SqlExecutor, timeout int64) ([]sdk.V2WorkflowRunJob, error) {
	ctx, next := telemetry.Span(ctx, "workflow_v2.LoadOldWaitingRunJob")
	defer next()
	query := gorpmapping.NewQuery(`
    SELECT *
    FROM v2_workflow_run_job
    WHERE status = $1 AND now() - queued > $2 * INTERVAL '1' SECOND
    LIMIT 100
    `).Args(sdk.StatusWaiting, timeout)
	return getAllRunJobs(ctx, db, query)
}

func LoadDeadJobs(ctx context.Context, db gorp.SqlExecutor) ([]sdk.V2WorkflowRunJob, error) {
	query := gorpmapping.NewQuery(`
    SELECT v2_workflow_run_job.*
		FROM v2_workflow_run_job
		LEFT JOIN v2_worker ON v2_worker.run_job_id = v2_workflow_run_job.id
		WHERE v2_workflow_run_job.status = $1 AND v2_worker.id IS NULL
    ORDER BY started
    LIMIT 100
  `).Args(sdk.StatusBuilding)
	return getAllRunJobs(ctx, db, query)
}

func CountRunJobsByProjectStatusAndRegions(ctx context.Context, db gorp.SqlExecutor, pkeys []string, statusFilter []sdk.V2WorkflowRunJobStatus, regionsFilter []string) (int64, error) {
	var statusStrings []string
	for _, v := range statusFilter {
		statusStrings = append(statusStrings, string(v))
	}
	query := `
    SELECT count(v2_workflow_run_job.*)
		FROM v2_workflow_run_job
		WHERE 
		  (array_length($1::text[], 1) IS NULL OR v2_workflow_run_job.project_key = ANY($1))
		  AND
      v2_workflow_run_job.status = ANY($2)
			AND
			(array_length($3::text[], 1) IS NULL OR v2_workflow_run_job.region = ANY($3))
	`
	count, err := db.SelectInt(query, pq.StringArray(pkeys), pq.StringArray(statusStrings), pq.StringArray(regionsFilter))
	return count, sdk.WithStack(err)
}

func LoadRunJobsByProjectStatusAndRegions(ctx context.Context, db gorp.SqlExecutor, pkeys []string, statusFilter []sdk.V2WorkflowRunJobStatus, regionsFilter []string, offset int, limit int) ([]sdk.V2WorkflowRunJob, error) {
	var statusStrings []string
	for _, v := range statusFilter {
		statusStrings = append(statusStrings, string(v))
	}
	query := gorpmapping.NewQuery(`
    SELECT v2_workflow_run_job.*
		FROM v2_workflow_run_job
		WHERE 
		  (array_length($1::text[], 1) IS NULL OR v2_workflow_run_job.project_key = ANY($1))
			AND
		  v2_workflow_run_job.status = ANY($2)
			AND
		  (array_length($3::text[], 1) IS NULL OR v2_workflow_run_job.region = ANY($3))
		ORDER BY queued ASC
		OFFSET $4 LIMIT $5
  `).Args(pq.StringArray(pkeys), pq.StringArray(statusStrings), pq.StringArray(regionsFilter), offset, limit)
	return getAllRunJobs(ctx, db, query)
}
