-- +migrate Up
ALTER TABLE v2_workflow_run_job ADD COLUMN "gate_inputs" TEXT;

-- +migrate Down
ALTER TABLE v2_workflow_run_job DROP COLUMN "gate_inputs";
