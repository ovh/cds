-- +migrate Up
ALTER TABLE workflow_node_join_source DROP CONSTRAINT fk_workflow_node_join_source_node;
SELECT create_foreign_key_idx_cascade('fk_workflow_node_join_source_node', 'workflow_node_join_source', 'workflow_node', 'workflow_node_id', 'id');

-- +migrate Down;
ALTER TABLE workflow_node_join_source DROP CONSTRAINT fk_workflow_node_join_source_node;
SELECT create_foreign_key('FK_WORKFLOW_NODE_JOIN_SOURCE_NODE', 'workflow_node_join_source', 'workflow_node', 'workflow_node_id', 'id');