-- +migrate Up
ALTER TABLE "workflow_project_integration" ADD COLUMN config JSONB;
ALTER TABLE "integration_model" ADD COLUMN additional_default_config JSONB;
UPDATE integration_model SET additional_default_config = deployment_default_config;

-- +migrate Down
ALTER TABLE "workflow_project_integration" DROP COLUMN config;
