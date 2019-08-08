-- +migrate Up
ALTER TABLE integration_model ADD COLUMN event BOOLEAN NOT NULL DEFAULT false;

CREATE TABLE IF NOT EXISTS workflow_project_integration (
  id BIGSERIAL PRIMARY KEY,
  workflow_id BIGINT,
  project_integration_id BIGINT
);
SELECT create_foreign_key_idx_cascade('FK_WORKFLOW_PROJECT_INTEGRATION_WORKFLOW', 'workflow_project_integration', 'workflow', 'workflow_id', 'id');
SELECT create_foreign_key_idx_cascade('FK_WORKFLOW_PROJECT_INTEGRATION_PROJECT_INTEGRATION', 'workflow_project_integration', 'project_integration', 'project_integration_id', 'id');
select create_unique_index('workflow_project_integration', 'IDX_WORKFLOW_PROJECT_INTEGRATION_PROJECT_INTEGRATION_ID_WORKFLOW_ID', 'workflow_id,project_integration_id');

-- +migrate Down
DROP TABLE workflow_project_integration;
ALTER TABLE integration_model DROP COLUMN event;
