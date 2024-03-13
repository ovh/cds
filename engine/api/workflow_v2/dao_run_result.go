package workflow_v2

import (
	"context"

	"github.com/go-gorp/gorp"
	"github.com/lib/pq"
	"github.com/rockbears/log"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/sdk"
)

func InsertRunResult(ctx context.Context, db gorp.SqlExecutor, runResult *sdk.V2WorkflowRunResult) error {
	entity := dbV2WorkflowRunResult{*runResult}
	if err := gorpmapping.Insert(db, &entity); err != nil {
		return err
	}
	log.Info(ctx, "run result %+v inserted", runResult)
	return nil
}

func UpdateRunResult(ctx context.Context, db gorp.SqlExecutor, runResult *sdk.V2WorkflowRunResult) error {
	entity := dbV2WorkflowRunResult{*runResult}
	return gorpmapping.Update(db, &entity)
}

func LoadRunResults(ctx context.Context, db gorp.SqlExecutor, runID string, runAttempts int64) ([]sdk.V2WorkflowRunResult, error) {
	results, err := loadRunResultsByRunIDs(ctx, db, runID)
	if err != nil {
		return nil, err
	}
	if results, has := results[runID]; has {
		var runResults []sdk.V2WorkflowRunResult
		for _, r := range results {
			if r.RunAttempt == runAttempts {
				runResults = append(runResults, r.V2WorkflowRunResult)
			}
		}
		return runResults, nil
	}
	return nil, nil
}

func LoadAbandonnedRunResultsID(ctx context.Context, db gorp.SqlExecutor) ([]string, error) {
	query := `
  	select v2_workflow_run_result.id 
	from v2_workflow_run_result 
	join v2_workflow_run_job on v2_workflow_run_job.id = v2_workflow_run_result.workflow_run_job_id
	where v2_workflow_run_job.status IN ('Fail', 'Stopped')
	and v2_workflow_run_result.status = 'PENDING'
	order by v2_workflow_run_result.issued_at ASC`
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
	return &result.V2WorkflowRunResult, nil
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
	return &result.V2WorkflowRunResult, nil
}

func LoadRunResultsByRunJobID(ctx context.Context, db gorp.SqlExecutor, runJobID string) ([]sdk.V2WorkflowRunResult, error) {
	query := gorpmapping.NewQuery(`select * from v2_workflow_run_result where workflow_run_job_id = $1`).Args(runJobID)
	var result []dbV2WorkflowRunResult
	if err := gorpmapping.GetAll(ctx, db, query, &result); err != nil {
		return nil, sdk.WrapError(err, "unable to load run results for run job %s", runJobID)
	}
	runResults := make([]sdk.V2WorkflowRunResult, 0, len(result))
	for _, r := range result {
		runResults = append(runResults, r.V2WorkflowRunResult)
	}
	return runResults, nil
}
