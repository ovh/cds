-- +migrate Up
ALTER TABLE "workflow_project_integration" ADD COLUMN config JSONB;
ALTER TABLE "integration_model" RENAME COLUMN deployment_default_config TO additional_default_config;

-- +migrate Down
ALTER TABLE "workflow_project_integration" DROP COLUMN config;
ALTER TABLE "integration_model" RENAME COLUMN additional_default_config TO deployment_default_config;



