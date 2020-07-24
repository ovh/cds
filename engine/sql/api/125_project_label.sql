-- +migrate Up
CREATE TABLE IF NOT EXISTS project_label (
  id BIGSERIAL PRIMARY KEY,
  project_id BIGINT,
  name VARCHAR(100),
  color VARCHAR(100)
);
SELECT create_foreign_key_idx_cascade('FK_PROJECT_LABEL_PROJECT', 'project_label', 'project', 'project_id', 'id');
SELECT create_unique_index('project_label', 'IDX_PROJECT_LABEL_PROJECT_ID_NAME', 'project_id,name');

CREATE TABLE IF NOT EXISTS project_label_workflow (
  label_id BIGINT,
  workflow_id BIGINT,
  PRIMARY KEY(label_id, workflow_id)
);
SELECT create_foreign_key_idx_cascade('FK_PROJECT_LABEL_WORKFLOW_WORKFLOW', 'project_label_workflow', 'workflow', 'workflow_id', 'id');
SELECT create_foreign_key_idx_cascade('FK_PROJECT_LABEL_WORKFLOW_PROJECT_LABEL', 'project_label_workflow', 'project_label', 'label_id', 'id');

-- +migrate Down
DROP TABLE project_label;
DROP TABLE project_label_workflow;
