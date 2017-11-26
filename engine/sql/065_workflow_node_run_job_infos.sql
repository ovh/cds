-- +migrate Up
CREATE TABLE IF NOT EXISTS "workflow_node_run_job_info" (
    id BIGSERIAL PRIMARY KEY,
    workflow_node_run_job_id BIGINT,
    workflow_node_run_id BIGINT,
    created TIMESTAMP WITH TIME ZONE,
    spawninfos JSONB
);

SELECT create_foreign_key_idx_cascade('FK_WORKFLOW_NODE_RUN_INFOS_WORKFLOW_NODE_RUN_JOB', 'workflow_node_run_job_info', 'workflow_node_run_job', 'workflow_node_run_job_id', 'id');

ALTER TABLE workflow_node_run_job DROP COLUMN spawninfos;

-- +migrate Down
DROP TABLE workflow_node_run_job_info;
ALTER TABLE workflow_node_run_job ADD COLUMN spawninfos JSONB;