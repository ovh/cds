-- +migrate Up
ALTER TABLE v2_workflow_run_job ADD COLUMN "matrix" TEXT;

-- +migrate Down
ALTER TABLE v2_workflow_run_job DROP COLUMN "matrix";
