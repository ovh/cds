-- +migrate Up

ALTER table platform_model RENAME TO integration_model;
ALTER table project_platform RENAME TO project_integration;
ALTER TABLE grpc_plugin RENAME COLUMN platform_model_id TO integration_model_id;
ALTER TABLE project_integration RENAME COLUMN platform_model_id TO integration_model_id;
ALTER TABLE application_deployment_strategy RENAME COLUMN project_platform_id TO project_integration_id;
ALTER TABLE workflow_node_context RENAME COLUMN project_platform_id TO project_integration_id;

-- +migrate Down

ALTER TABLE grpc_plugin RENAME COLUMN integration_model_id TO platform_model_id;
ALTER TABLE project_integration RENAME COLUMN integration_model_id TO platform_model_id;
ALTER TABLE application_deployment_strategy RENAME COLUMN project_integration_id TO project_platform_id;
ALTER TABLE workflow_node_context RENAME COLUMN project_integration_id TO project_platform_id;
ALTER table project_integration RENAME TO project_platform;
ALTER table integration_model RENAME TO platform_model;
