-- +migrate Up
ALTER TABLE v2_workflow_run_job ADD COLUMN "matrix" TEXT;
DROP INDEX idx_workflow_run_job_unq;

-- +migrate Down
ALTER TABLE v2_workflow_run_job DROP COLUMN "matrix";
