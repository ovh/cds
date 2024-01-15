-- +migrate Up
ALTER TABLE v2_workflow_run_result ADD COLUMN "artifact_manager_integration_id" BIGINT;
SELECT create_foreign_key('FK_V2_WORKFLOW_RUN_RESULT_PROJECT_INTEGRATION', 'v2_workflow_run_result', 'project_integration', 'artifact_manager_integration_id', 'id');

-- +migrate Down
ALTER TABLE v2_workflow_run_result DROP COLUMN "artifact_manager_integration_id";
