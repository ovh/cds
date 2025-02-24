package workflow_v2

import (
	"context"
	"database/sql"

	"github.com/go-gorp/gorp"
	"github.com/lib/pq"
	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/sdk"
)

type ConcurrencyRule struct {
	MinPool int64  `db:"min"`
	Order   string `db:"order"`
	Cancel  bool   `db:"cancel_in_progress"`
}

func CountRunningRunJobWithWorkflowConcurrency(ctx context.Context, db gorp.SqlExecutor, proj string, vcs string, repo string, workflow string, concurrencyName string) (int64, error) {
	q := `SELECT count(id)
	FROM v2_workflow_run_job 
	WHERE 
		project_key = $1 AND 
		vcs_server = $2 AND 
		repository = $3 AND 
		workflow_name = $4 AND 
		concurrency->>'name' = $5 AND
		concurrency->>'scope' = $6 AND
		status = ANY($7)`
	nb, err := db.SelectInt(q, proj, vcs, repo, workflow, concurrencyName, sdk.V2RunJobConcurrencyScopeWorkflow, pq.StringArray{string(sdk.V2WorkflowRunJobStatusWaiting), string(sdk.V2WorkflowRunJobStatusScheduling), string(sdk.V2WorkflowRunJobStatusBuilding)})
	if err != nil {
		return 0, sdk.WithStack(err)
	}
	return nb, nil
}

func CountBlockedRunJobWithWorkflowConcurrency(ctx context.Context, db gorp.SqlExecutor, proj string, vcs string, repo string, workflow string, concurrencyName string) (int64, error) {
	q := `SELECT count(id)
	FROM v2_workflow_run_job 
	WHERE 
		project_key = $1 AND 
		vcs_server = $2 AND 
		repository = $3 AND 
		workflow_name = $4 AND 
		concurrency->>'name' = $5 AND
		concurrency->>'scope' = $6 AND
		status = $7`
	nb, err := db.SelectInt(q, proj, vcs, repo, workflow, concurrencyName, sdk.V2RunJobConcurrencyScopeWorkflow, sdk.V2WorkflowRunJobStatusBlocked)
	if err != nil {
		return 0, sdk.WithStack(err)
	}
	return nb, nil
}

func LoadConcurrencyRules(ctx context.Context, db gorp.SqlExecutor, proj string, vcs string, repo string, workflow string, concurrencyName string) ([]ConcurrencyRule, error) {
	q := `SELECT concurrency->>'order' as order, concurrency->>'cancel_in_progress' as cancel, min(concurrency->>'pool') as min
	FROM v2_workflow_run_job 
	WHERE 
		project_key = $1 AND 
		vcs_server = $2 AND 
		repository = $3 AND 
		workflow_name = $4 AND 
		concurrency->>'name' = $5 AND
		concurrency->>'scope' = $6
		status = ANY($7)
	GROUP BY concurrency->>'order'`

	var rules []ConcurrencyRule
	if _, err := db.Select(&rules, q, proj, vcs, repo, workflow, concurrencyName, sdk.V2RunJobConcurrencyScopeWorkflow, pq.StringArray{string(sdk.V2WorkflowRunJobStatusBlocked), string(sdk.V2WorkflowRunJobStatusWaiting), string(sdk.V2WorkflowRunJobStatusScheduling), string(sdk.V2WorkflowRunJobStatusBuilding)}); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
	}
	if len(rules) == 1 && (rules[0].MinPool == 0 && rules[0].Order == "") {
		return nil, nil
	}
	return rules, nil
}

////////////

func LoadOldestRunJobWithSameConcurrencyOnSameWorkflow(ctx context.Context, db gorp.SqlExecutor, proj string, vcs string, repo string, workflow string, concurrencyName string) (*sdk.V2WorkflowRunJob, error) {
	q := `SELECT * 
		FROM v2_workflow_run_job 
		WHERE project_key = $1 AND 
			vcs_server = $2 AND 
			repository = $3 AND 
			workflow_name = $4 AND 
			concurrency->>'name' = $5 AND
			concurrency->>'scope' = $6 AND
			status = $7
	ORDER BY queued ASC LIMIT 1`
	query := gorpmapping.NewQuery(q).Args(proj, vcs, repo, workflow, concurrencyName, sdk.V2RunJobConcurrencyScopeWorkflow, sdk.V2WorkflowRunJobStatusBlocked)
	return getRunJob(ctx, db, query)
}

func LoadNewestRunJobWithSameConcurrencyOnSameWorkflow(ctx context.Context, db gorp.SqlExecutor, proj string, vcs string, repo string, workflow string, concurrencyName string) (*sdk.V2WorkflowRunJob, error) {
	q := `SELECT * 
		FROM v2_workflow_run_job 
		WHERE project_key = $1 AND 
			vcs_server = $2 AND 
			repository = $3 AND 
			workflow_name = $4 AND 
			concurrency->>'name' = $5 AND
			concurrency->>'scope' = $6 AND 
			status = $7
	ORDER BY queued DESC LIMIT 1`
	query := gorpmapping.NewQuery(q).Args(proj, vcs, repo, workflow, concurrencyName, sdk.V2RunJobConcurrencyScopeWorkflow, sdk.V2WorkflowRunJobStatusBlocked)
	return getRunJob(ctx, db, query)
}
