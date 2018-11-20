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
  environments JSONB,
  parameters JSONB
);

SELECT create_unique_index('workflow_template', 'IDX_WORKFLOW_TEMPLATES_KEY_NAME', 'group_id,slug');

SELECT create_foreign_key_idx_cascade('FK_WORKFLOW_TEMPLATE_GROUP', 'workflow_template', 'group', 'group_id', 'id');

CREATE TABLE IF NOT EXISTS "workflow_template_instance" (
  id BIGSERIAL PRIMARY KEY,
  workflow_template_id BIGINT NOT NULL,
  project_id BIGINT NOT NULL,
  workflow_id BIGINT,
  workflow_template_version BIGINT NOT NULL,
  request JSONB
);

SELECT create_foreign_key_idx_cascade('FK_WORKFLOW_TEMPLATE_INSTANCE_TEMPLATE', 'workflow_template_instance', 'workflow_template', 'workflow_template_id', 'id');
SELECT create_foreign_key_idx_cascade('FK_WORKFLOW_TEMPLATE_INSTANCE_PROJECT', 'workflow_template_instance', 'project', 'project_id', 'id');
SELECT create_foreign_key_idx_cascade('FK_WORKFLOW_TEMPLATE_INSTANCE_WORKFLOW', 'workflow_template_instance', 'workflow', 'workflow_id', 'id');

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

CREATE TABLE IF NOT EXISTS workflow_template_instance_audit (
  id BIGSERIAL PRIMARY KEY,
  triggered_by VARCHAR(100),
  created TIMESTAMP WITH TIME ZONE,
  data_before TEXT,
  data_after TEXT,
  event_type VARCHAR(100),
  data_type VARCHAR(20),
  workflow_template_instance_id BIGINT
);

SELECT create_foreign_key_idx_cascade('FK_WORKFLOW_TEMPLATE_INSTANCE_AUDIT', 'workflow_template_instance_audit', 'workflow_template_instance', 'workflow_template_instance_id', 'id');

-- +migrate Down

DROP TABLE IF EXISTS workflow_template_instance_audit;
DROP TABLE IF EXISTS workflow_template_audit;
DROP TABLE IF EXISTS workflow_template_instance;
DROP TABLE IF EXISTS workflow_template;
