-- +migrate Up
ALTER TABLE v2_workflow_run_job ADD COLUMN "matrix" TEXT;
select create_unique_index('v2_workflow_run_job', 'IDX_workflow_run_job_unq', 'workflow_run_id,job_id,matrix,run_number,run_attempt');

-- +migrate Down
ALTER TABLE v2_workflow_run_job DROP COLUMN "matrix";
select create_unique_index('v2_workflow_run_job', 'IDX_workflow_run_job_unq', 'workflow_run_id,job_id,run_number,run_attempt');

