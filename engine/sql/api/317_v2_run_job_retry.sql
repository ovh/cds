
-- +migrate Up
ALTER TABLE v2_workflow_run_job ADD COLUMN retry NUMERIC default 0;
drop index idx_workflow_run_job_unq;
select create_unique_index('v2_workflow_run_job', 'idx_workflow_run_job_unq', 'workflow_run_id,job_id,matrix,run_number,run_attempt,retry');


-- +migrate Down
drop index idx_workflow_run_job_unq;
ALTER TABLE v2_workflow_run_job DROP COLUMN retry;
select create_unique_index('v2_workflow_run_job', 'idx_workflow_run_job_unq', 'workflow_run_id,job_id,matrix,run_number,run_attempt');


