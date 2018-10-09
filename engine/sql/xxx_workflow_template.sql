-- +migrate Up

CREATE TABLE IF NOT EXISTS "workflow_template" (
  id BIGSERIAL PRIMARY KEY,
  group_id BIGINT NOT NULL,
  name TEXT NOT NULL,
  version BIGINT NOT NULL,
  value TEXT,
  pipelines JSONB,
  parameters JSONB
);

SELECT create_index('workflow_template','IDX_WORKFLOW_TEMPLATES_GROUP_ID', 'group_id');
SELECT create_unique_index('workflow_template', 'IDX_WORKFLOW_TEMPLATES_KEY_NAME', 'group_id,name');

CREATE TABLE IF NOT EXISTS "workflow_template_workflow" (
  id BIGSERIAL PRIMARY KEY,
  workflow_template_id BIGINT NOT NULL,
  workflow_id BIGINT NOT NULL,
  workflow_template_version BIGINT NOT NULL,
  request JSONB
);

SELECT create_unique_index('workflow_template_workflow', 'IDX_WORKFLOW_TEMPLATES_WORKFLOW_KEY_ID', 'workflow_template_id,workflow_id');

-- +migrate Down

DROP TABLE IF EXISTS workflow_template;
DROP TABLE IF EXISTS workflow_template_workflow;
