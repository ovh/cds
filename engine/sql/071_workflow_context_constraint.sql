-- +migrate Up
ALTER TABLE workflow_node_context DROP CONSTRAINT fk_workflow_node_application;
ALTER TABLE workflow_node_context DROP CONSTRAINT fk_workflow_node_environment;
ALTER TABLE workflow_node_context ADD CONSTRAINT fk_workflow_node_application FOREIGN KEY (application_id) REFERENCES application(id) ON DELETE SET NULL;
ALTER TABLE workflow_node_context ADD CONSTRAINT fk_workflow_node_environment FOREIGN KEY (environment_id) REFERENCES environment(id) ON DELETE SET NULL;

-- +migrate Down
ALTER TABLE workflow_node_context DROP CONSTRAINT fk_workflow_node_application;
ALTER TABLE workflow_node_context DROP CONSTRAINT fk_workflow_node_environment;
SELECT create_foreign_key_idx_cascade('FK_WORKFLOW_NODE_APPLICATION', 'workflow_node_context', 'application', 'application_id', 'id');
SELECT create_foreign_key_idx_cascade('FK_WORKFLOW_NODE_ENVIRONMENT', 'workflow_node_context', 'environment', 'environment_id', 'id');