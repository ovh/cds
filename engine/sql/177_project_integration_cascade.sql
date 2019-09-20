-- +migrate Up

ALTER TABLE workflow_node_context DROP CONSTRAINT "fk_workflow_node_project_integration";
SELECT create_foreign_key_idx_cascade('FK_WORKFLOW_NODE_PROJECT_INTEGRATION', 'workflow_node_context', 'project_integration', 'project_integration_id', 'id');

-- +migrate Down
SELECT 1;
