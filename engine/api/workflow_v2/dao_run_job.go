package workflow_v2

import (
	"context"
	"time"

	"github.com/go-gorp/gorp"
	"github.com/lib/pq"
	"github.com/rockbears/log"
	"go.opencensus.io/trace"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
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
		jobRuns = append(jobRuns, rj.V2WorkflowRunJob)
	}
	return jobRuns, nil
}

func getRunJob(ctx context.Context, db gorp.SqlExecutor, query gorpmapping.Query) (*sdk.V2WorkflowRunJob, error) {
	var dbWkfRunJob dbWorkflowRunJob
	found, err := gorpmapping.Get(ctx, db, query, &dbWkfRunJob)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, sdk.ErrNotFound
	}
	isValid, err := gorpmapping.CheckSignature(dbWkfRunJob, dbWkfRunJob.Signature)
	if err != nil {
		return nil, err
	}
	if !isValid {
		log.Error(ctx, "run job %s on run %s: data corrupted", dbWkfRunJob.ID, dbWkfRunJob.WorkflowRunID)
		return nil, sdk.ErrNotFound
	}
	return &dbWkfRunJob.V2WorkflowRunJob, nil
}

func InsertRunJob(ctx context.Context, db gorpmapper.SqlExecutorWithTx, wrj *sdk.V2WorkflowRunJob) error {
	ctx, next := telemetry.Span(ctx, "workflow_v2.InsertRunJob", trace.StringAttribute(telemetry.TagJob, wrj.JobID))
	defer next()
	wrj.ID = sdk.UUID()
	wrj.Queued = time.Now()
	dbWkfRunJob := &dbWorkflowRunJob{V2WorkflowRunJob: *wrj}

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
	if err := gorpmapping.UpdateAndSign(ctx, db, dbWkfRunJob); err != nil {
		return err
	}
	*wrj = dbWkfRunJob.V2WorkflowRunJob
	return nil
}

func LoadRunJobsByRunID(ctx context.Context, db gorp.SqlExecutor, runID string) ([]sdk.V2WorkflowRunJob, error) {
	ctx, next := telemetry.Span(ctx, "LoadRunJobsByRunID")
	defer next()
	query := gorpmapping.NewQuery("SELECT * from v2_workflow_run_job WHERE workflow_run_id = $1").Args(runID)
	return getAllRunJobs(ctx, db, query)
}

func LoadRunJobByID(ctx context.Context, db gorp.SqlExecutor, jobRunID string) (*sdk.V2WorkflowRunJob, error) {
	ctx, next := telemetry.Span(ctx, "workflow_v2.LoadRunJobByID")
	defer next()
	query := gorpmapping.NewQuery("SELECT * from v2_workflow_run_job WHERE id = $1").Args(jobRunID)
	return getRunJob(ctx, db, query)
}

func LoadRunJobByName(ctx context.Context, db gorp.SqlExecutor, wrID string, jobName string) (*sdk.V2WorkflowRunJob, error) {
	ctx, next := telemetry.Span(ctx, "workflow_v2.LoadRunJobByID")
	defer next()
	query := gorpmapping.NewQuery("SELECT * from v2_workflow_run_job WHERE workflow_run_id = $1 AND job_id = $2").Args(wrID, jobName)
	return getRunJob(ctx, db, query)
}

func LoadQueuedRunJobByModelTypeAndRegion(ctx context.Context, db gorp.SqlExecutor, regionName string, modelType string) ([]sdk.V2WorkflowRunJob, error) {
	ctx, next := telemetry.Span(ctx, "workflow_v2.LoadQueuedRunJobByModelTypeAndRegion")
	defer next()
	query := gorpmapping.NewQuery("SELECT * from v2_workflow_run_job WHERE status = $1 AND model_type = $2 and region = $3 ORDER BY queued ASC").
		Args(sdk.StatusWaiting, modelType, regionName)
	return getAllRunJobs(ctx, db, query)
}

func LoadRunJobsByRunIDAndStatus(ctx context.Context, db gorp.SqlExecutor, runID string, status []string) ([]sdk.V2WorkflowRunJob, error) {
	ctx, next := telemetry.Span(ctx, "workflow_v2.LoadRunJobsByRunIDAndStatus")
	defer next()
	query := gorpmapping.NewQuery("SELECT * from v2_workflow_run_job WHERE workflow_run_id = $1 AND status = ANY($2)").Args(runID, pq.StringArray(status))
	return getAllRunJobs(ctx, db, query)
}
