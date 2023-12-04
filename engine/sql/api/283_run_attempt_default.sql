-- +migrate Up
ALTER TABLE v2_workflow_run ALTER COLUMN run_attempt SET DEFAULT 1;
ALTER TABLE v2_workflow_run_job ALTER COLUMN run_attempt SET DEFAULT 1;


-- +migrate Down
SELECT 1;
