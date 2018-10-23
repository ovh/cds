-- +migrate Up

CREATE TABLE IF NOT EXISTS "workflow_template" (
  id BIGSERIAL PRIMARY KEY,
  group_id BIGINT NOT NULL,
  name TEXT NOT NULL,
  slug TEXT NOT NULL,
  description TEXT,
  version BIGINT NOT NULL,
  value TEXT,
  pipelines JSONB,
  applications JSONB,
  parameters JSONB
);

SELECT create_index('workflow_template','IDX_WORKFLOW_TEMPLATES_GROUP_ID', 'group_id');
SELECT create_unique_index('workflow_template', 'IDX_WORKFLOW_TEMPLATES_KEY_NAME', 'group_id,slug');

CREATE TABLE IF NOT EXISTS "workflow_template_instance" (
  id BIGSERIAL PRIMARY KEY,
  workflow_template_id BIGINT NOT NULL,
  project_id BIGINT NOT NULL,
  workflow_id BIGINT,
  workflow_template_version BIGINT NOT NULL,
  request JSONB
);

CREATE TABLE IF NOT EXISTS workflow_template_audit (
  id BIGSERIAL PRIMARY KEY,
  triggered_by VARCHAR(100),
  created TIMESTAMP WITH TIME ZONE,
  data_before TEXT,
  data_after TEXT,
  event_type VARCHAR(100),
  data_type VARCHAR(20),
  workflow_template_id BIGINT
);

SELECT create_foreign_key_idx_cascade('FK_WORKFLOW_TEMPLATE_AUDIT', 'workflow_template_audit', 'workflow_template', 'workflow_template_id', 'id');

-- +migrate Down

DROP TABLE IF EXISTS workflow_template_audit;
DROP TABLE IF EXISTS workflow_template_instance;
DROP TABLE IF EXISTS workflow_template;
