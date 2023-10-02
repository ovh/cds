-- +migrate Up
CREATE TABLE v2_workflow_hook (
  id              uuid  PRIMARY KEY,
  project_key     VARCHAR(255) NOT NULL,
  vcs_name        VARCHAR(255) NOT NULL,
  repository_name VARCHAR(255) NOT NULL,
  entity_id       uuid NOT NULL,
  workflow_name   VARCHAR(255) NOT NULL,
  branch          VARCHAR(255) NOT NULL,
  commit          VARCHAR(255) NOT NULL,
  type            VARCHAR(255) NOT NULL,
  data            JSONB NOT NULL,
  "sig"           BYTEA,
  "signer"        TEXT
);

SELECT create_foreign_key_idx_cascade('fk_v2_workflow_hook_project', 'v2_workflow_hook', 'project', 'project_key', 'projectkey');
SELECT create_foreign_key_idx_cascade('fk_v2_workflow_hook_entity', 'v2_workflow_hook', 'entity', 'entity_id', 'id');
CREATE INDEX idx_v2_workflow_hook_repository ON v2_workflow_hook(type, (data->>'vcs_server'), (data->>'repository_name'), (data->>'repository_event'));
CREATE INDEX idx_v2_workflow_hook_model ON v2_workflow_hook(type, (data->>'model'));
CREATE INDEX idx_v2_workflow_hook_workflow ON v2_workflow_hook(type, project_key, vcs_name, repository_name, workflow_name);

-- +migrate Down
DROP TABLE v2_workflow_hook;
