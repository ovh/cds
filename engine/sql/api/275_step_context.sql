-- +migrate Up
ALTER TABLE v2_workflow_run_job DROP COLUMN "steps_context";
ALTER TABLE v2_workflow_run_job ADD COLUMN "steps_status" TEXT;

-- +migrate Down
ALTER TABLE v2_workflow_run_job DROP COLUMN "steps_status";
ALTER TABLE v2_workflow_run_job ADD COLUMN "steps_context" TEXT;

