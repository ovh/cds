-- +migrate Up
CREATE TABLE IF NOT EXISTS workflow_label (
  id BIGSERIAL PRIMARY KEY,
  project_id BIGINT,
  name VARCHAR(100),
  color VARCHAR(100)
);
SELECT create_foreign_key_idx_cascade('FK_WORKFLOW_LABEL_PROJECT', 'workflow_label', 'project', 'project_id', 'id');
SELECT create_unique_index('workflow_label', 'IDX_WORKFLOW_LABEL_PROJECT_ID_NAME', 'project_id,name');

CREATE TABLE IF NOT EXISTS workflow_label_workflow (
  label_id BIGINT,
  workflow_id BIGINT,
  PRIMARY KEY(label_id, workflow_id)
);
SELECT create_foreign_key_idx_cascade('FK_WORKFLOW_LABEL_WORKFLOW_WORKFLOW', 'workflow_label_workflow', 'workflow', 'workflow_id', 'id');
SELECT create_foreign_key_idx_cascade('FK_WORKFLOW_LABEL_WORKFLOW_LABEL', 'workflow_label_workflow', 'workflow_label', 'label_id', 'id');

-- +migrate Down
DROP TABLE workflow_label;
DROP TABLE workflow_label_workflow;
