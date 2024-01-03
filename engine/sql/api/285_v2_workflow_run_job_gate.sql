-- +migrate Up
ALTER TABLE v2_workflow_run_job ADD COLUMN "gate_inputs" TEXT;
ALTER TABLE v2_workflow_run ADD COLUMN "job_event" JSONB;

-- +migrate Down
ALTER TABLE v2_workflow_run_job DROP COLUMN "gate_inputs";
ALTER TABLE v2_workflow_run DROP COLUMN "job_event";
