-- The commit and workflow filters of a run search are matched against expressions, which only an
-- index describes and only its statistics measure.

-- +migrate Up notransaction

-- CONCURRENTLY, since runs are written to this table constantly. It cannot go through
-- create_index(): a function always runs inside a transaction, which the concurrent build refuses.
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_v2_workflow_run_git_sha
	ON v2_workflow_run ((contexts -> 'git' ->> 'sha'));

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_v2_workflow_run_project_workflow_started
	ON v2_workflow_run (project_key, ((vcs_server || '/' || repository || '/' || workflow_name)), started);

-- +migrate Down
DROP INDEX IF EXISTS idx_v2_workflow_run_git_sha;
DROP INDEX IF EXISTS idx_v2_workflow_run_project_workflow_started;
