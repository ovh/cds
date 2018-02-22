-- +migrate Up
select create_index('workflow_node_run_job_logs', 'IDX_WORKFLOW_LOG_STEP', 'workflow_node_run_job_id,step_order');


-- +migrate Down
DROP INDEX IDX_WORKFLOW_LOG_STEP;