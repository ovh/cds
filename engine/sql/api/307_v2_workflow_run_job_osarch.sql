-- +migrate Up
ALTER TABLE v2_workflow_run_job ADD COLUMN "model_osarch" VARCHAR(255);
UPDATE v2_workflow_run_job set model_osarch = 'linux/amd64';

-- +migrate Down
ALTER TABLE v2_workflow_run_job DROP COLUMN "model_osarch";
