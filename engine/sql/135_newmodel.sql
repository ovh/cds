-- +migrate Up
ALTER TABLE workflow ADD COLUMN workflow_data JSONB;
ALTER TABLE workflow_run ADD COLUMN version INT DEFAULT 1;

CREATE TABLE w_node (
  id BIGSERIAL PRIMARY KEY,
  workflow_id BIGINT,
  trigger_id BIGINT,
  name VARCHAR(100),
  ref VARCHAR(100),
  type VARCHAR(100)
);

SELECT create_foreign_key_idx_cascade('FK_W_NODE_WORKFLOW', 'w_node', 'workflow', 'workflow_id', 'id');

CREATE TABLE w_node_hook (
  id BIGSERIAL PRIMARY KEY,
  uuid VARCHAR(255),
  ref VARCHAR(50),
  node_id BIGINT,
  hook_model_id BIGINT,
  config JSONB
);

SELECT create_foreign_key_idx_cascade('FK_W_NODE_HOOK_NODE', 'w_node_hook', 'w_node', 'node_id', 'id');
SELECT create_foreign_key_idx_cascade('FK_W_NODE_HOOK_MODEL', 'w_node_hook', 'workflow_hook_model', 'hook_model_id', 'id');

CREATE TABLE w_node_context (
  id BIGSERIAL PRIMARY KEY,
  node_id BIGINT,
  pipeline_id BIGINT,
  application_id BIGINT,
  environment_id BIGINT,
  project_platform_id BIGINT,
  default_payload JSONB,
  default_pipeline_parameters JSONB,
  conditions JSONB,
  mutex BOOLEAN
);

SELECT create_foreign_key_idx_cascade('FK_W_NODE_CONTEXT_NODE', 'w_node_context', 'w_node', 'node_id', 'id');
SELECT create_foreign_key_idx_cascade('FK_W_NODE_CONTEXT_PIPELINE', 'w_node_context', 'pipeline', 'pipeline_id', 'id');
SELECT create_foreign_key_idx_cascade('FK_W_NODE_CONTEXT_APPLICATION', 'w_node_context', 'application', 'application_id', 'id');
SELECT create_foreign_key_idx_cascade('FK_W_NODE_CONTEXT_ENVIRONMENT', 'w_node_context', 'environment', 'environment_id', 'id');
SELECT create_foreign_key_idx_cascade('FK_W_NODE_CONTEXT_PLATFORM', 'w_node_context', 'project_platform', 'project_platform_id', 'id');


CREATE TABLE w_node_trigger (
  id BIGSERIAL PRIMARY KEY,
  parent_node_id BIGINT,
  child_node_id BIGINT
);

SELECT create_foreign_key_idx_cascade('FK_W_NODE_TRIGGER_PARENT', 'w_node_trigger', 'w_node', 'parent_node_id', 'id');
SELECT create_foreign_key('FK_W_NODE_TRIGGER_CHILD', 'w_node_trigger', 'w_node', 'child_node_id', 'id');

ALTER TABLE w_node
ADD CONSTRAINT FK_NODE_TRIGGER_ID
FOREIGN KEY(trigger_id) REFERENCES w_node_trigger(id) ON DELETE CASCADE DEFERRABLE INITIALLY DEFERRED;

CREATE TABLE w_node_outgoing_hook (
  id BIGSERIAL PRIMARY KEY,
  node_id BIGINT,
  hook_model_id BIGINT,
  config JSONB
);

SELECT create_foreign_key_idx_cascade('FK_W_NODE_OUTGOING_HOOK_NODE', 'w_node_outgoing_hook', 'w_node', 'node_id', 'id');
SELECT create_foreign_key('FK_W_NODE_OUTGOING_HOOK_MODEL', 'w_node_outgoing_hook', 'workflow_outgoing_hook_model', 'hook_model_id', 'id');

CREATE TABLE w_node_join (
  id BIGSERIAL PRIMARY KEY,
  node_id BIGINT,
  parent_id BIGINT
);

SELECT create_foreign_key_idx_cascade('FK_W_NODE_JOIN_NODE', 'w_node_join', 'w_node', 'node_id', 'id');
SELECT create_foreign_key_idx_cascade('FK_W_NODE_JOIN_PARENT', 'w_node_join', 'w_node', 'parent_id', 'id');

ALTER TABLE workflow_node_run ADD COLUMN uuid VARCHAR(255);
ALTER TABLE workflow_node_run ADD COLUMN outgoinghook JSONB;
ALTER TABLE workflow_node_run ADD COLUMN hook_execution_timestamp VARCHAR(20);
ALTER TABLE workflow_node_run ADD COLUMN execution_id VARCHAR(255);
ALTER TABLE workflow_node_run ADD COLUMN callback JSONB;

-- +migrate Down
ALTER TABLE workflow DROP COLUMN workflow_data;
DROP TABLE w_node CASCADE;
DROP TABLE w_node_hook CASCADE;
DROP TABLE w_node_context CASCADE;
DROP TABLE w_node_trigger CASCADE;
DROP TABLE w_node_outgoing_hook CASCADE;
DROP TABLE w_node_join CASCADE;

ALTER TABLE workflow_node_run DROP COLUMN uuid;
ALTER TABLE workflow_node_run DROP COLUMN outgoinghook;
ALTER TABLE workflow_node_run DROP COLUMN hook_execution_timestamp;
ALTER TABLE workflow_node_run DROP COLUMN execution_id;
ALTER TABLE workflow_node_run DROP COLUMN callback;
