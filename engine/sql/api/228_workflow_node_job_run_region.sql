-- +migrate Up
ALTER TABLE "workflow_node_run_job" ADD COLUMN IF NOT EXISTS "region" VARCHAR(256);
SELECT create_unique_index('workflow_node_run_job','IDX_WORKFLOW_NODE_RUN_JOB_REGION','region');

-- +migrate Down
DROP INDEX IF EXISTS idx_workflow_node_run_job_region;
ALTER TABLE "workflow_node_run_job" DROP COLUMN IF EXISTS "region";
