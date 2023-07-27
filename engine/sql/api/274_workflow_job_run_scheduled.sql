-- +migrate Up
ALTER TABLE v2_workflow_run_job ADD COLUMN "scheduled" TIMESTAMP WITH TIME ZONE DEFAULT LOCALTIMESTAMP;
ALTER TABLE v2_worker RENAME COLUMN job_run_id TO run_job_id;

-- +migrate Down
ALTER TABLE v2_workflow_run_job DROP COLUMN "scheduled";
ALTER TABLE v2_worker RENAME COLUMN run_job_id TO job_run_id;

