-- +migrate Up
ALTER TABLE platform_model ADD COLUMN platform_model_plugin JSONB;

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

-- +migrate Down
ALTER TABLE worker_model DROP COLUMN platform_model_plugin;
DROP TABLE application_deployment_strategy;
ALTER TABLE workflow_node_context DROP COLUMN project_platform_id;