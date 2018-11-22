-- +migrate Up

ALTER TABLE workflow_node_outgoing_hook DROP CONSTRAINT FK_WORKFLOW_NODE_HOOK_WORKFLOW_OUTGOING_HOOK_MODEL;
SELECT create_foreign_key('FK_WORKFLOW_NODE_HOOK_WORKFLOW_OUTGOING_HOOK_MODEL', 'workflow_node_outgoing_hook', 'workflow_outgoing_hook_model', 'workflow_hook_model_id', 'id');


-- +migrate Down
SELECT 1;
