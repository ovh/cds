-- +migrate Up
ALTER TABLE v2_workflow_run_job DROP COLUMN outputs;

-- +migrate Down
ALTER TABLE v2_workflow_run_job ADD COLUMN outputs TEXT;
