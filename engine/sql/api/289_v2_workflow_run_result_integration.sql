-- +migrate Up
ALTER TABLE v2_workflow_run_result ADD COLUMN "artifact_manager_integration_name" VARCHAR(100);

-- +migrate Down
ALTER TABLE v2_workflow_run_result DROP COLUMN "artifact_manager_integration_name";
