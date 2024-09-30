package workflow_v2

import (
	"context"

	"github.com/go-gorp/gorp"
	"github.com/lib/pq"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/telemetry"
)

func getAllRunResults(ctx context.Context, db gorp.SqlExecutor, query gorpmapping.Query) ([]sdk.V2WorkflowRunResult, error) {
	var dbWkfRunResults []dbV2WorkflowRunResult
	if err := gorpmapping.GetAll(ctx, db, query, &dbWkfRunResults); err != nil {
		return nil, err
	}
	jobResults := make([]sdk.V2WorkflowRunResult, 0, len(dbWkfRunResults))
	for i := range dbWkfRunResults {
		rr := &dbWkfRunResults[i].V2WorkflowRunResult
		rr.ComputedFields()
		jobResults = append(jobResults, *rr)
	}
	return jobResults, nil
}

func InsertRunResult(ctx context.Context, db gorp.SqlExecutor, runResult *sdk.V2WorkflowRunResult) error {
	entity := dbV2WorkflowRunResult{*runResult}
	if err := gorpmapping.Insert(db, &entity); err != nil {
		return err
	}
	runResult.ComputedFields()
	return nil
}

func UpdateRunResult(ctx context.Context, db gorp.SqlExecutor, runResult *sdk.V2WorkflowRunResult) error {
	entity := dbV2WorkflowRunResult{*runResult}
	runResult.ComputedFields()
	return gorpmapping.Update(db, &entity)
}

func LoadRunResultsByRunIDAttempt(ctx context.Context, db gorp.SqlExecutor, runID string, runAttempt int64) ([]sdk.V2WorkflowRunResult, error) {
	ctx, next := telemetry.Span(ctx, "LoadRunResultsByRunIDAttempt")
	defer next()
	query := gorpmapping.NewQuery(`
    SELECT *
    FROM v2_workflow_run_result
    WHERE workflow_run_id = $1 AND run_attempt = $2
    ORDER BY issued_at ASC
	`).Args(runID, runAttempt)
	return getAllRunResults(ctx, db, query)
}

func LoadRunResultsByRunID(ctx context.Context, db gorp.SqlExecutor, runID string) ([]sdk.V2WorkflowRunResult, error) {
	ctx, next := telemetry.Span(ctx, "LoadRunResultsByRunID")
	defer next()
	query := gorpmapping.NewQuery(`
    SELECT *
    FROM v2_workflow_run_result
    WHERE workflow_run_id = $1
    ORDER BY issued_at ASC
	`).Args(runID)
	return getAllRunResults(ctx, db, query)
}

func LoadAbandonnedRunResultsID(ctx context.Context, db gorp.SqlExecutor) ([]string, error) {
	query := `
    SELECT v2_workflow_run_result.id 
    FROM v2_workflow_run_result 
    JOIN v2_workflow_run_job ON v2_workflow_run_job.id = v2_workflow_run_result.workflow_run_job_id
    WHERE v2_workflow_run_job.status IN ('Fail', 'Stopped') AND v2_workflow_run_result.status = 'PENDING'
    ORDER BY v2_workflow_run_result.issued_at ASC
	`
	var results pq.StringArray
	if _, err := db.Select(&results, query); err != nil {
		return nil, sdk.WrapError(err, "unable to load abandonned run results")
	}
	return results, nil
}

func LoadRunResult(ctx context.Context, db gorp.SqlExecutor, runID string, id string) (*sdk.V2WorkflowRunResult, error) {
	query := gorpmapping.NewQuery(`select * from v2_workflow_run_result where id = $1 AND workflow_run_id = $2`).Args(id, runID)
	var result dbV2WorkflowRunResult
	found, err := gorpmapping.Get(ctx, db, query, &result)
	if err != nil {
		return nil, sdk.WrapError(err, "unable to load run result %v", id)
	}
	if !found {
		return nil, sdk.WrapError(sdk.ErrNotFound, "unable to run load result id=%s workflow_run_id=%s", id, runID)
	}
	r := &result.V2WorkflowRunResult
	r.ComputedFields()
	return r, nil
}

func LoadAndLockRunResultByID(ctx context.Context, db gorp.SqlExecutor, id string) (*sdk.V2WorkflowRunResult, error) {
	query := gorpmapping.NewQuery(`select * from v2_workflow_run_result where id = $1 for update skip locked`).Args(id)
	var result dbV2WorkflowRunResult
	found, err := gorpmapping.Get(ctx, db, query, &result)
	if err != nil {
		return nil, sdk.WrapError(err, "unable to load run result %v", id)
	}
	if !found {
		return nil, nil
	}
	r := &result.V2WorkflowRunResult
	r.ComputedFields()
	return r, nil
}

func LoadRunResultsByRunJobID(ctx context.Context, db gorp.SqlExecutor, runJobID string) ([]sdk.V2WorkflowRunResult, error) {
	ctx, next := telemetry.Span(ctx, "LoadRunResultsByRunJobID")
	defer next()
	query := gorpmapping.NewQuery(`
    SELECT *
    FROM v2_workflow_run_result
    WHERE workflow_run_job_id = $1
	`).Args(runJobID)
	return getAllRunResults(ctx, db, query)
}
