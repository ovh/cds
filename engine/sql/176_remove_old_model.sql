-- +migrate Up
DROP TABLE workflow_node_join_source;
DROP TABLE workflow_node_join_trigger CASCADE;
DROP TABLE workflow_node_fork_trigger CASCADE;
DROP TABLE workflow_node_outgoing_hook_trigger CASCADE;
DROP TABLE workflow_node_trigger CASCADE;
DROP TABLE workflow_node_context;
DROP TABLE workflow_node_hook;
DROP TABLE workflow_node_join;
DROP TABLE workflow_node_fork;
DROP TABLE workflow_node_outgoing_hook;
DROP TABLE workflow_node;

-- +migrate Down
SELECT 1;
