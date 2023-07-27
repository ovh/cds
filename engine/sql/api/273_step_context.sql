-- +migrate Up
ALTER TABLE v2_workflow_run_job ADD COLUMN "steps_context" TEXT;

-- +migrate Down
ALTER TABLE v2_workflow_run_job DROP COLUMN "steps_context";

