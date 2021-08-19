-- +migrate Up
DROP INDEX IF EXISTS idx_workflow_node_run_job_region;
SELECT create_index('workflow_node_run_job','IDX_WORKFLOW_NODE_RUN_JOB_REGION','region');

-- +migrate Down
DROP INDEX IF EXISTS idx_workflow_node_run_job_region;
