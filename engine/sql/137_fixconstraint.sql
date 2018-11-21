-- +migrate Up
ALTER TABLE w_node_context DROP CONSTRAINT FK_W_NODE_CONTEXT_PIPELINE;
ALTER TABLE w_node_context DROP CONSTRAINT FK_W_NODE_CONTEXT_APPLICATION;
ALTER TABLE w_node_context DROP CONSTRAINT FK_W_NODE_CONTEXT_ENVIRONMENT;
ALTER TABLE w_node_context DROP CONSTRAINT FK_W_NODE_CONTEXT_PLATFORM;

SELECT create_foreign_key('FK_W_NODE_CONTEXT_PIPELINE', 'w_node_context', 'pipeline', 'pipeline_id', 'id');
SELECT create_foreign_key('FK_W_NODE_CONTEXT_APPLICATION', 'w_node_context', 'application', 'application_id', 'id');
SELECT create_foreign_key('FK_W_NODE_CONTEXT_ENVIRONMENT', 'w_node_context', 'environment', 'environment_id', 'id');
SELECT create_foreign_key('FK_W_NODE_CONTEXT_PLATFORM', 'w_node_context', 'project_platform', 'project_platform_id', 'id');

-- +migrate Down
ALTER TABLE w_node_context DROP CONSTRAINT FK_W_NODE_CONTEXT_PIPELINE;
ALTER TABLE w_node_context DROP CONSTRAINT FK_W_NODE_CONTEXT_APPLICATION;
ALTER TABLE w_node_context DROP CONSTRAINT FK_W_NODE_CONTEXT_ENVIRONMENT;
ALTER TABLE w_node_context DROP CONSTRAINT FK_W_NODE_CONTEXT_PLATFORM;

SELECT create_foreign_key_idx_cascade('FK_W_NODE_CONTEXT_PIPELINE', 'w_node_context', 'pipeline', 'pipeline_id', 'id');
SELECT create_foreign_key_idx_cascade('FK_W_NODE_CONTEXT_APPLICATION', 'w_node_context', 'application', 'application_id', 'id');
SELECT create_foreign_key_idx_cascade('FK_W_NODE_CONTEXT_ENVIRONMENT', 'w_node_context', 'environment', 'environment_id', 'id');
SELECT create_foreign_key_idx_cascade('FK_W_NODE_CONTEXT_PLATFORM', 'w_node_context', 'project_platform', 'project_platform_id', 'id');
