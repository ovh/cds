-- +migrate Up
DROP INDEX idx_workflow_node_run_subnum;
SELECT create_unique_index('workflow_node_run', 'idx_workflow_node_run_subnum', 'workflow_run_id,workflow_node_id, num, sub_num');

-- +migrate Down
SELECT 1;
