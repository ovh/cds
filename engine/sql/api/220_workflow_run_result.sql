-- +migrate Up
CREATE TABLE "workflow_run_result" (
    id VARCHAR(36) PRIMARY KEY, -- technical ID
    created TIMESTAMP WITH TIME ZONE, -- creation date
    workflow_run_id BIGINT NOT NULL,
    workflow_node_run_id BIGINT NOT NULL,
    workflow_run_job_id BIGINT NOT NULL,
    sub_num BIGINT NOT NULL,
    type VARCHAR(255) NOT NULL,
    data JSONB NOT NULL
);

SELECT create_foreign_key_idx_cascade('FK_workflow_run_result_run_id', 'workflow_run_result', 'workflow_run', 'workflow_run_id', 'id');
SELECT create_foreign_key_idx_cascade('FK_workflow_run_result_node_run_id', 'workflow_run_result', 'workflow_node_run', 'workflow_node_run_id', 'id');

SELECT create_index('workflow_run_result', 'idx_workflow_run_result_node_run_sub_num_type', 'workflow_node_run_id,sub_num,type');

-- +migrate Down
DROP TABLE "workflow_run_result";
