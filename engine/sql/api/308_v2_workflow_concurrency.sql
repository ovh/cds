-- +migrate Up
ALTER TABLE v2_workflow_run_job ADD COLUMN "concurrency" JSONB;


-- +migrate Down
ALTER TABLE v2_workflow_run_job DROP COLUMN "concurrency";
