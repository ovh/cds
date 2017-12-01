-- +migrate Up
CREATE TABLE IF NOT EXISTS "workflow_run_artifacts" (
  id bigserial NOT NULL,
  name text,
  tag text,
  workflow_id BIGINT NOT NULL,
  workflow_node_run_id BIGINT NOT NULL,
  download_hash text,
  size bigint,
  perm integer,
  md5sum text,
  object_path text
);

SELECT create_foreign_key_idx_cascade('FK_WORKFLOW_RUN_WORKFLOW', 'workflow_run_artifacts', 'workflow_run', 'workflow_id', 'id');
SELECT create_foreign_key_idx_cascade('FK_WORKFLOW_RUN_PROJECT', 'workflow_run_artifacts', 'workflow_node_run', 'workflow_node_run_id', 'id');
SELECT create_index('workflow_run_artifacts', 'IDX_WORKFLOW_ARTIFACT_WORKFLOW_ID', 'workflow_id');
SELECT create_index('workflow_run_artifacts', 'IDX_WORKFLOW_ARTIFACT_WORKFLOW_NODE_RUN_ID', 'workflow_node_run_id');

-- +migrate Down
DROP TABLE workflow_run_artifacts;