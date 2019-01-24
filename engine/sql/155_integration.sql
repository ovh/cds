-- +migrate Up

CREATE TABLE integration_model AS TABLE platform_model;
SELECT create_unique_index('integration_model', 'IDX_INTEGRATION_MODEL_NAME', 'name');
SELECT create_primary_key('integration_model', 'id');

CREATE TABLE project_integration AS TABLE project_platform;
ALTER TABLE project_integration RENAME COLUMN platform_model_id TO integration_model_id;
SELECT create_primary_key('project_integration', 'id');

SELECT create_foreign_key_idx_cascade('fk_project_integration', 'project_integration', 'project', 'project_id', 'id');
SELECT create_unique_index('project_integration', 'IDX_PROJECT_INTEGRATION_NAME', 'project_id,name');
SELECT create_index('project_integration', 'IDX_PROJECT_INTEGRATION', 'project_id,name');

ALTER TABLE grpc_plugin ADD COLUMN integration_model_id BIGINT;
UPDATE grpc_plugin set integration_model_id = platform_model_id;

ALTER TABLE application_deployment_strategy ADD COLUMN project_integration_id BIGINT;
UPDATE application_deployment_strategy set project_integration_id = project_platform_id;
SELECT create_foreign_key_idx_cascade('fk_application_deployment_strategy_integration', 'application_deployment_strategy', 'project_integration', 'project_integration_id', 'id');

ALTER TABLE workflow_node_context ADD COLUMN project_integration_id BIGINT;
UPDATE workflow_node_context set project_integration_id = project_platform_id;
SELECT create_foreign_key('FK_WORKFLOW_NODE_PROJECT_INTEGRATION', 'workflow_node_context', 'project_integration', 'project_integration_id', 'id');

ALTER TABLE w_node_context ADD COLUMN project_integration_id BIGINT;
UPDATE w_node_context set project_integration_id = project_platform_id;
SELECT create_foreign_key_idx_cascade('FK_W_NODE_CONTEXT_INTEGRATION', 'w_node_context', 'project_integration', 'project_integration_id', 'id');

ALTER TABLE workflow_node_run_job ADD COLUMN integration_plugin_binaries JSONB;
UPDATE workflow_node_run_job set integration_plugin_binaries = platform_plugin_binaries;

-- +migrate Down

DROP TABLE integration_model;
DROP TABLE project_integration;

ALTER TABLE grpc_plugin DROP COLUMN integration_model_id;
ALTER TABLE application_deployment_strategy DROP COLUMN project_integration_id;
ALTER TABLE workflow_node_context DROP COLUMN project_integration_id;
ALTER TABLE w_node_context DROP COLUMN project_integration_id;
ALTER TABLE workflow_node_run_job DROP COLUMN integration_plugin_binaries;
