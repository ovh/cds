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
	MinPool int64  `db:"pool"`
	Order   string `db:"order"`
	Cancel  bool   `db:"cancel"`
}

func CountRunningRunJobWithProjectConcurrency(ctx context.Context, db gorp.SqlExecutor, proj string, concurrencyName string) (int64, error) {
	q := `SELECT count(id)
	FROM v2_workflow_run_job 
	WHERE 
		project_key = $1 AND 
		concurrency->>'name' = $2 AND
		concurrency->>'scope' = $3 AND
		status = ANY($4)`
	nb, err := db.SelectInt(q, proj, concurrencyName, sdk.V2RunJobConcurrencyScopeProject, pq.StringArray{string(sdk.V2WorkflowRunJobStatusWaiting), string(sdk.V2WorkflowRunJobStatusScheduling), string(sdk.V2WorkflowRunJobStatusBuilding)})
	if err != nil {
		return 0, sdk.WithStack(err)
	}
	return nb, nil
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

func CountBlockedRunJobWithProjectConcurrency(ctx context.Context, db gorp.SqlExecutor, proj string, concurrencyName string) (int64, error) {
	q := `SELECT count(id)
	FROM v2_workflow_run_job 
	WHERE 
		project_key = $1 AND 
		concurrency->>'name' = $2 AND
		concurrency->>'scope' = $3 AND
		status = $4`
	nb, err := db.SelectInt(q, proj, concurrencyName, sdk.V2RunJobConcurrencyScopeProject, sdk.V2WorkflowRunJobStatusBlocked)
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

func LoadProjectConcurrencyRules(ctx context.Context, db gorp.SqlExecutor, proj string, concurrencyName string) ([]ConcurrencyRule, error) {
	q := `SELECT concurrency->>'order' as order, concurrency->>'cancel-in-progress' as cancel, min(concurrency->>'pool') as pool
	FROM v2_workflow_run_job 
	WHERE 
		project_key = $1 AND 
		concurrency->>'name' = $2 AND
		concurrency->>'scope' = $3 AND
		status = ANY($4)
	GROUP BY concurrency->>'order', concurrency->>'cancel-in-progress'`

	var rules []ConcurrencyRule
	if _, err := db.Select(&rules, q, proj, concurrencyName, sdk.V2RunJobConcurrencyScopeProject, pq.StringArray{string(sdk.V2WorkflowRunJobStatusBlocked), string(sdk.V2WorkflowRunJobStatusWaiting), string(sdk.V2WorkflowRunJobStatusScheduling), string(sdk.V2WorkflowRunJobStatusBuilding)}); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, sdk.WithStack(err)
	}
	if len(rules) == 1 && (rules[0].MinPool == 0 && rules[0].Order == "") {
		return nil, nil
	}
	return rules, nil
}

func LoadWorkflowConcurrencyRules(ctx context.Context, db gorp.SqlExecutor, proj string, vcs string, repo string, workflow string, concurrencyName string) ([]ConcurrencyRule, error) {
	q := `SELECT concurrency->>'order' as order, concurrency->>'cancel-in-progress' as cancel, min(concurrency->>'pool') as pool
	FROM v2_workflow_run_job 
	WHERE 
		project_key = $1 AND 
		vcs_server = $2 AND 
		repository = $3 AND 
		workflow_name = $4 AND 
		concurrency->>'name' = $5 AND
		concurrency->>'scope' = $6 AND
		status = ANY($7)
	GROUP BY concurrency->>'order', concurrency->>'cancel-in-progress'`

	var rules []ConcurrencyRule
	if _, err := db.Select(&rules, q, proj, vcs, repo, workflow, concurrencyName, sdk.V2RunJobConcurrencyScopeWorkflow, pq.StringArray{string(sdk.V2WorkflowRunJobStatusBlocked), string(sdk.V2WorkflowRunJobStatusWaiting), string(sdk.V2WorkflowRunJobStatusScheduling), string(sdk.V2WorkflowRunJobStatusBuilding)}); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, sdk.WithStack(err)
	}
	if len(rules) == 1 && (rules[0].MinPool == 0 && rules[0].Order == "") {
		return nil, nil
	}
	return rules, nil
}

func LoadOldestRunJobWithWorkflowScopedConcurrency(ctx context.Context, db gorp.SqlExecutor, proj string, vcs string, repo string, workflow string, concurrencyName string, status []string, limit int64) ([]sdk.V2WorkflowRunJob, error) {
	q := `SELECT * 
		FROM v2_workflow_run_job 
		WHERE project_key = $1 AND 
			vcs_server = $2 AND 
			repository = $3 AND 
			workflow_name = $4 AND 
			concurrency->>'name' = $5 AND
			concurrency->>'scope' = $6 AND
			status = ANY($7)
	ORDER BY run_number ASC, queued ASC LIMIT $8`
	query := gorpmapping.NewQuery(q).Args(proj, vcs, repo, workflow, concurrencyName, sdk.V2RunJobConcurrencyScopeWorkflow, pq.StringArray(status), limit)
	return getAllRunJobs(ctx, db, query)
}

func LoadNewestRunJobWithWorkflowScopedConcurrency(ctx context.Context, db gorp.SqlExecutor, proj string, vcs string, repo string, workflow string, concurrencyName string, status []string, limit int64) ([]sdk.V2WorkflowRunJob, error) {
	q := `SELECT * 
		FROM v2_workflow_run_job 
		WHERE project_key = $1 AND 
			vcs_server = $2 AND 
			repository = $3 AND 
			workflow_name = $4 AND 
			concurrency->>'name' = $5 AND
			concurrency->>'scope' = $6 AND 
			status = ANY($7)
	ORDER BY run_number DESC, queued DESC`
	args := []interface{}{proj, vcs, repo, workflow, concurrencyName, sdk.V2RunJobConcurrencyScopeWorkflow, pq.StringArray(status)}
	if limit > 0 {
		q += ` LIMIT $8`
		args = append(args, limit)
	}

	query := gorpmapping.NewQuery(q).Args(args...)
	return getAllRunJobs(ctx, db, query)
}

func LoadOldestRunJobWithProjectScopedConcurrency(ctx context.Context, db gorp.SqlExecutor, proj string, concurrencyName string, status []string, limit int64) ([]sdk.V2WorkflowRunJob, error) {

	q := `SELECT * 
		FROM v2_workflow_run_job 
		WHERE project_key = $1 AND 
			concurrency->>'name' = $2 AND
			concurrency->>'scope' = $3 AND
			status = ANY($4)
	ORDER BY queued ASC LIMIT $5`
	query := gorpmapping.NewQuery(q).Args(proj, concurrencyName, sdk.V2RunJobConcurrencyScopeProject, pq.StringArray(status), limit)
	return getAllRunJobs(ctx, db, query)
}

func LoadNewestRunJobWithProjectScopedConcurrency(ctx context.Context, db gorp.SqlExecutor, proj string, concurrencyName string, status []string, limit int64) ([]sdk.V2WorkflowRunJob, error) {
	q := `SELECT * 
		FROM v2_workflow_run_job 
		WHERE project_key = $1 AND 
			concurrency->>'name' = $2 AND
			concurrency->>'scope' = $3 AND 
			status = ANY($4)
	ORDER BY queued DESC`
	args := []interface{}{proj, concurrencyName, sdk.V2RunJobConcurrencyScopeProject, pq.StringArray(status)}
	if limit > 0 {
		q += ` LIMIT $5`
		args = append(args, limit)
	}

	query := gorpmapping.NewQuery(q).Args().Args(args...)
	return getAllRunJobs(ctx, db, query)
}
