-- +migrate Up
ALTER TABLE workflow_node_run DROP CONSTRAINT fk_workflow_node_run_workflow_node;

-- +migrate Down
SELECT create_foreign_key('FK_WORKFLOW_NODE_RUN_WORKFLOW_NODE', 'workflow_node_run', 'workflow_node', 'workflow_node_id', 'id');
