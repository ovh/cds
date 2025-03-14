package workflow_v2

import (
	"context"
	"database/sql"

	"github.com/go-gorp/gorp"
	"github.com/lib/pq"
	"github.com/ovh/cds/sdk"
)

type ConcurrencyRule struct {
	MinPool int64  `db:"pool"`
	Order   string `db:"order"`
	Cancel  bool   `db:"cancel"`
}

const (
	ConcurrencyObjectTypeWorkflow ConcurrencyObjectType = "WORKFLOW"
	ConcurrencyObjectTypeJob      ConcurrencyObjectType = "JOB"
)

type ConcurrencyObjectType string
type ConcurrencyObject struct {
	Type ConcurrencyObjectType `db:"type"`
	ID   string                `db:"id"`
}

func CountRunningWithProjectConcurrency(ctx context.Context, db gorp.SqlExecutor, proj string, concurrencyName string) (int64, error) {
	q := `WITH jobs AS 
	(
		SELECT count(id) as nb
		FROM v2_workflow_run_job 
		WHERE 
			project_key = $1 AND 
			concurrency->>'name' = $2 AND
			concurrency->>'scope' = $3 AND
			status = ANY($4)
	), runs AS (
		SELECT count(id) as nb
		FROM v2_workflow_run
		WHERE 
			project_key = $1 AND 
			concurrency->>'name' = $2 AND
			concurrency->>'scope' = $3 AND
			status = ANY($4)
	) 
	SELECT jobs.nb + runs.nb
	FROM jobs, runs`
	nb, err := db.SelectInt(q, proj, concurrencyName, sdk.V2RunConcurrencyScopeProject, pq.StringArray{string(sdk.V2WorkflowRunJobStatusWaiting), string(sdk.V2WorkflowRunJobStatusScheduling), string(sdk.V2WorkflowRunJobStatusBuilding)})
	if err != nil {
		return 0, sdk.WithStack(err)
	}
	return nb, nil
}

func CountRunningWithWorkflowConcurrency(ctx context.Context, db gorp.SqlExecutor, proj string, vcs string, repo string, workflow string, concurrencyName string) (int64, error) {
	q := `WITH jobs as (
		SELECT count(id) as nb
		FROM v2_workflow_run_job 
		WHERE 
			project_key = $1 AND 
			vcs_server = $2 AND 
			repository = $3 AND 
			workflow_name = $4 AND 
			concurrency->>'name' = $5 AND
			concurrency->>'scope' = $6 AND
			status = ANY($7)
	), runs as (
		SELECT count(id) as nb
		FROM v2_workflow_run 
		WHERE 
			project_key = $1 AND 
			vcs_server = $2 AND 
			repository = $3 AND 
			workflow_name = $4 AND 
			concurrency->>'name' = $5 AND
			concurrency->>'scope' = $6 AND
			status = ANY($7)
	) SELECT jobs.nb + runs.nb FROM jobs, runs`
	nb, err := db.SelectInt(q, proj, vcs, repo, workflow, concurrencyName, sdk.V2RunConcurrencyScopeWorkflow, pq.StringArray{string(sdk.V2WorkflowRunJobStatusWaiting), string(sdk.V2WorkflowRunJobStatusScheduling), string(sdk.V2WorkflowRunJobStatusBuilding)})
	if err != nil {
		return 0, sdk.WithStack(err)
	}
	return nb, nil
}

func CountBlockedRunWithProjectConcurrency(ctx context.Context, db gorp.SqlExecutor, proj string, concurrencyName string) (int64, error) {
	q := `WITH jobs as (
		SELECT count(id) as nb
		FROM v2_workflow_run_job 
		WHERE 
			project_key = $1 AND 
			concurrency->>'name' = $2 AND
			concurrency->>'scope' = $3 AND
			status = $4
	), runs as (
		SELECT count(id) as nb
		FROM v2_workflow_run
		WHERE 
			project_key = $1 AND 
			concurrency->>'name' = $2 AND
			concurrency->>'scope' = $3 AND
			status = $4
	) SELECT jobs.nb + runs.nb FROM jobs, runs`
	nb, err := db.SelectInt(q, proj, concurrencyName, sdk.V2RunConcurrencyScopeProject, sdk.V2WorkflowRunJobStatusBlocked)
	if err != nil {
		return 0, sdk.WithStack(err)
	}
	return nb, nil
}

func CountBlockedWithWorkflowConcurrency(ctx context.Context, db gorp.SqlExecutor, proj string, vcs string, repo string, workflow string, concurrencyName string) (int64, error) {
	q := `WITH jobs as (
		SELECT count(id) as nb
		FROM v2_workflow_run_job 
		WHERE 
			project_key = $1 AND 
			vcs_server = $2 AND 
			repository = $3 AND 
			workflow_name = $4 AND 
			concurrency->>'name' = $5 AND
			concurrency->>'scope' = $6 AND
			status = $7
	), runs as (
		SELECT count(id) as nb
		FROM v2_workflow_run 
		WHERE 
			project_key = $1 AND 
			vcs_server = $2 AND 
			repository = $3 AND 
			workflow_name = $4 AND 
			concurrency->>'name' = $5 AND
			concurrency->>'scope' = $6 AND
			status = $7
	) SELECT jobs.nb + runs.nb FROM jobs, runs`
	nb, err := db.SelectInt(q, proj, vcs, repo, workflow, concurrencyName, sdk.V2RunConcurrencyScopeWorkflow, sdk.V2WorkflowRunJobStatusBlocked)
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
	GROUP BY concurrency->>'order', concurrency->>'cancel-in-progress'
	UNION
	SELECT concurrency->>'order' as order, concurrency->>'cancel-in-progress' as cancel, min(concurrency->>'pool') as pool
	FROM v2_workflow_run
	WHERE 
		project_key = $1 AND 
		concurrency->>'name' = $2 AND
		concurrency->>'scope' = $3 AND
		status = ANY($4)
	GROUP BY concurrency->>'order', concurrency->>'cancel-in-progress'
	`

	var rules []ConcurrencyRule
	if _, err := db.Select(&rules, q, proj, concurrencyName, sdk.V2RunConcurrencyScopeProject, pq.StringArray{string(sdk.V2WorkflowRunJobStatusBlocked), string(sdk.V2WorkflowRunJobStatusWaiting), string(sdk.V2WorkflowRunJobStatusScheduling), string(sdk.V2WorkflowRunJobStatusBuilding)}); err != nil {
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
	GROUP BY concurrency->>'order', concurrency->>'cancel-in-progress'
	UNION
	SELECT concurrency->>'order' as order, concurrency->>'cancel-in-progress' as cancel, min(concurrency->>'pool') as pool
	FROM v2_workflow_run 
	WHERE 
		project_key = $1 AND 
		vcs_server = $2 AND 
		repository = $3 AND 
		workflow_name = $4 AND 
		concurrency->>'name' = $5 AND
		concurrency->>'scope' = $6 AND
		status = ANY($7)
	GROUP BY concurrency->>'order', concurrency->>'cancel-in-progress'
	`

	var rules []ConcurrencyRule
	if _, err := db.Select(&rules, q, proj, vcs, repo, workflow, concurrencyName, sdk.V2RunConcurrencyScopeWorkflow, pq.StringArray{string(sdk.V2WorkflowRunJobStatusBlocked), string(sdk.V2WorkflowRunJobStatusWaiting), string(sdk.V2WorkflowRunJobStatusScheduling), string(sdk.V2WorkflowRunJobStatusBuilding)}); err != nil {
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

func LoadOldestRunJobWithWorkflowScopedConcurrency(ctx context.Context, db gorp.SqlExecutor, proj string, vcs string, repo string, workflow string, concurrencyName string, rjStatus []string, workflowRunStatus sdk.V2WorkflowRunStatus, limit int64) ([]ConcurrencyObject, error) {
	q := `WITH jobs as (
		SELECT id, queued as last_modified, 'JOB' as type 
		FROM v2_workflow_run_job 
		WHERE project_key = $1 AND 
			vcs_server = $2 AND 
			repository = $3 AND 
			workflow_name = $4 AND 
			concurrency->>'name' = $5 AND
			concurrency->>'scope' = $6 AND
			status = ANY($7)
		ORDER BY last_modified ASC LIMIT $9
	), runs as (
		SELECT id, last_modified, 'WORKFLOW' as type 
		FROM v2_workflow_run 
		WHERE project_key = $1 AND 
			vcs_server = $2 AND 
			repository = $3 AND 
			workflow_name = $4 AND 
			concurrency->>'name' = $5 AND
			concurrency->>'scope' = $6 AND
			status = $8
		ORDER BY run_number ASC, last_modified ASC LIMIT $9
	) SELECT id, type FROM (
	 	SELECT * FROM jobs
		UNION
		SELECT * FROM runs
	) tmpl
	ORDER BY last_modified ASC LIMIT $9`
	var cos []ConcurrencyObject
	if _, err := db.Select(&cos, q, proj, vcs, repo, workflow, concurrencyName, sdk.V2RunConcurrencyScopeWorkflow, pq.StringArray(rjStatus), workflowRunStatus, limit); err != nil {
		return nil, sdk.WithStack(err)
	}
	return cos, nil
}

func LoadNewestRunJobWithWorkflowScopedConcurrency(ctx context.Context, db gorp.SqlExecutor, proj string, vcs string, repo string, workflow string, concurrencyName string, rjStatus []string, workflowRunStatus sdk.V2WorkflowRunStatus, limit interface{}) ([]ConcurrencyObject, error) {
	q := `WITH jobs as (
		SELECT id, queued as last_modified, 'JOB' as type
		FROM v2_workflow_run_job 
		WHERE project_key = $1 AND 
			vcs_server = $2 AND 
			repository = $3 AND 
			workflow_name = $4 AND 
			concurrency->>'name' = $5 AND
			concurrency->>'scope' = $6 AND 
			status = ANY($7)
		ORDER BY last_modified DESC
		LIMIT $9
	), runs as (
		SELECT id, last_modified, 'WORKFLOW' as type
		FROM v2_workflow_run
		WHERE project_key = $1 AND 
			vcs_server = $2 AND 
			repository = $3 AND 
			workflow_name = $4 AND 
			concurrency->>'name' = $5 AND
			concurrency->>'scope' = $6 AND 
			status = $8
		ORDER BY run_number DESC, last_modified DESC
		LIMIT $9
	) SELECT id, type FROM (
	 	SELECT * FROM jobs
		UNION
		SELECT * FROM runs
	) tmp 
	ORDER BY last_modified DESC
	LIMIT $9`

	var cos []ConcurrencyObject
	if _, err := db.Select(&cos, q, proj, vcs, repo, workflow, concurrencyName, sdk.V2RunConcurrencyScopeWorkflow, pq.StringArray(rjStatus), workflowRunStatus, limit); err != nil {
		return nil, sdk.WithStack(err)
	}
	return cos, nil
}

func LoadOldestRunJobWithProjectScopedConcurrency(ctx context.Context, db gorp.SqlExecutor, proj string, concurrencyName string, rjStatus []string, workflowStatus sdk.V2WorkflowRunStatus, limit int64) ([]ConcurrencyObject, error) {
	q := `WITH jobs as (
		-- GET THE n OLDEST RUN JOBS
		SELECT id as id, queued as last_modified, 'JOB' as type 
		FROM v2_workflow_run_job 
		WHERE project_key = $1 AND 
			concurrency->>'name' = $2 AND
			concurrency->>'scope' = $3 AND
			status = ANY($4)
		ORDER BY last_modified ASC LIMIT $6
	), runs as (
	    -- GET THE n OLDEST WORKFLOW RUN 
	    SELECT id as id, last_modified as last_modified, 'WORKFLOW' as type 
		FROM v2_workflow_run
		WHERE project_key = $1 AND 
			concurrency->>'name' = $2 AND
			concurrency->>'scope' = $3 AND
			status = $5
		ORDER BY run_number ASC, last_modified ASC LIMIT $6
	) SELECT id, type FROM (
	 	SELECT * FROM jobs
		UNION
		SELECT * FROM runs
	) tmp ORDER BY last_modified ASC LIMIT $6`
	var cos []ConcurrencyObject
	if _, err := db.Select(&cos, q, proj, concurrencyName, sdk.V2RunConcurrencyScopeProject, pq.StringArray(rjStatus), workflowStatus, limit); err != nil {
		return nil, sdk.WithStack(err)
	}
	return cos, nil
}

func LoadNewestRunJobWithProjectScopedConcurrency(ctx context.Context, db gorp.SqlExecutor, proj string, concurrencyName string, rjStatus []string, workflowRunStatus sdk.V2WorkflowRunStatus, limit interface{}) ([]ConcurrencyObject, error) {
	q := `WITH jobs as (
		SELECT id,  queued as last_modified, 'JOB' as type
		FROM v2_workflow_run_job 
		WHERE project_key = $1 AND 
			concurrency->>'name' = $2 AND
			concurrency->>'scope' = $3 AND 
			status = ANY($4)
			ORDER BY last_modified DESC
			LIMIT $6
	), runs as (
		SELECT id, last_modified, 'WORKFLOW' as type
		FROM v2_workflow_run 
		WHERE project_key = $1 AND 
			concurrency->>'name' = $2 AND
			concurrency->>'scope' = $3 AND 
			status = $5
			ORDER BY run_number DESC, last_modified DESC
			LIMIT $6
	) SELECT id, type FROM (
		SELECT * from jobs
		UNION
		SELECT * from runs 
	) tmp ORDER BY last_modified DESC
	LIMIT $6`
	var cos []ConcurrencyObject
	if _, err := db.Select(&cos, q, proj, concurrencyName, sdk.V2RunConcurrencyScopeProject, pq.StringArray(rjStatus), workflowRunStatus, limit); err != nil {
		return nil, sdk.WithStack(err)
	}
	return cos, nil
}

func LoadProjectConccurencyRunObjects(ctx context.Context, db gorp.SqlExecutor, proj string, concurrencyName string) ([]sdk.ProjectConcurrencyRunObject, error) {
	q := `WITH jobs as (
		SELECT workflow_run_id as workflow_run_id, queued as last_modified, 'JOB' as type, workflow_name, run_number, job_id as job_name, status
		FROM v2_workflow_run_job 
		WHERE project_key = $1 AND 
			concurrency->>'name' = $2 AND
			concurrency->>'scope' = $3 AND
			status = ANY($4)
		ORDER BY last_modified ASC
	), runs as (
	    SELECT id as workflow_run_id, last_modified as last_modified, 'WORKFLOW' as type, workflow_name, run_number, '' as job_name, status
		FROM v2_workflow_run
		WHERE project_key = $1 AND 
			concurrency->>'name' = $2 AND
			concurrency->>'scope' = $3 AND
			status = ANY($5)
		ORDER BY run_number ASC, last_modified ASC
	) SELECT * FROM (
	 	SELECT * FROM jobs
		UNION
		SELECT * FROM runs
	) tmp ORDER BY last_modified ASC`

	jobStatus := []string{string(sdk.V2WorkflowRunJobStatusBlocked), string(sdk.V2WorkflowRunJobStatusWaiting), string(sdk.V2WorkflowRunJobStatusScheduling), string(sdk.V2WorkflowRunJobStatusBuilding)}
	runStatus := []string{string(sdk.V2WorkflowRunStatusBlocked), string(sdk.V2WorkflowRunStatusBuilding)}
	var pcr []sdk.ProjectConcurrencyRunObject
	if _, err := db.Select(&pcr, q, proj, concurrencyName, sdk.V2RunConcurrencyScopeProject, pq.StringArray(jobStatus), pq.StringArray(runStatus)); err != nil {
		return nil, sdk.WithStack(err)
	}
	return pcr, nil
}
