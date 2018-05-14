-- +migrate Up
ALTER TABLE platform_model ADD COLUMN grpc_plugin_id BIGINT;
ALTER TABLE platform_model ADD COLUMN deployment_default_config JSONB;

CREATE TABLE application_deployment_strategy (
  application_id BIGINT,
  project_platform_id BIGINT,
  config JSONB,
  PRIMARY KEY (application_id, project_platform_id)
);

SELECT create_foreign_key_idx_cascade('fk_application_deployment_strategy_application', 'application_deployment_strategy', 'application', 'application_id', 'id');
SELECT create_foreign_key_idx_cascade('fk_application_deployment_strategy_platform', 'application_deployment_strategy', 'project_platform', 'project_platform_id', 'id');

ALTER TABLE workflow_node_context ADD COLUMN project_platform_id BIGINT;
SELECT create_foreign_key('FK_WORKFLOW_NODE_PROJECT_PLATFORM', 'workflow_node_context', 'project_platform', 'project_platform_id', 'id');

CREATE TABLE grpc_plugin (
  	id BIGSERIAL PRIMARY KEY,
  	name VARCHAR(50),
	  type VARCHAR(50),
	  author VARCHAR(50),
  	description TEXT,
  	binaries JSONB
);

select create_unique_index('grpc_plugin', 'IDX_GRPC_PLUGIN_NAME', 'name');

SELECT create_foreign_key('FK_PLATFORM_MODEL_GRPC_PLUGIN', 'platform_model', 'grpc_plugin', 'grpc_plugin_id', 'id');

ALTER TABLE workflow_node_run_job ADD COLUMN platform_plugin_binaries JSONB;

-- +migrate Down
ALTER TABLE platform_model DROP COLUMN grpc_plugin_id;
ALTER TABLE platform_model DROP COLUMN deployment_default_config;
DROP TABLE application_deployment_strategy;
ALTER TABLE workflow_node_context DROP COLUMN project_platform_id;
ALTER TABLE workflow_node_run_job DROP COLUMN platform_plugin_binaries;
DROP TABLE grpc_plugin;
