-- +migrate Up
CREATE TABLE IF NOT EXISTS "workflow" (
    id BIGSERIAL PRIMARY KEY,
    project_id BIGINT NOT NULL,
    name TEXT NOT NULL,
    root_node_id BIGINT
);

SELECT create_index('workflow', 'IDX_WORKFLOW_NAME', 'name');
SELECT create_foreign_key('FK_WORKFLOW_PROJECT', 'workflow', 'project', 'project_id', 'id');
SELECT create_foreign_key('FK_WORKFLOW_ROOT_NODE', 'workflow', 'workflow_node', 'root_node_id', 'id');

CREATE TABLE IF NOT EXISTS "workflow_node" (
    id BIGSERIAL PRIMARY KEY,
    workflow_id BIGINT NOT NULL,
    pipeline_id BIGINT NOT NULL
);

SELECT create_foreign_key('FK_WORKFLOW_NODE_WORKFLOW', 'workflow_node', 'workflow', 'workflow_id', 'id');
SELECT create_foreign_key('FK_WORKFLOW_NODE_PIPELINE', 'workflow_node', 'pipeline', 'pipeline_id', 'id');

CREATE TABLE IF NOT EXISTS "workflow_node_trigger" (
    id BIGSERIAL PRIMARY KEY,
    workflow_node_id BIGINT NOT NULL,
    workflow_dest_node_id BIGINT NOT NULL,
    conditions JSONB
);

SELECT create_foreign_key('FK_WORKFLOW_TRIGGER_WORKFLOW_NODE', 'workflow_node_trigger', 'workflow_node', 'workflow_node_id', 'id');
SELECT create_foreign_key('FK_WORKFLOW_TRIGGER_WORKFLOW_NODE', 'workflow_node_trigger', 'workflow_node', 'workflow_dest_node_id', 'id');

CREATE TABLE IF NOT EXISTS "workflow_node_context" (
    id BIGSERIAL PRIMARY KEY,
    workflow_node_id BIGINT NOT NULL,
    application_id BIGINT,
    environment_id BIGINT
);

SELECT create_foreign_key('FK_WORKFLOW_CONTEXT_WORKFLOW_NODE', 'workflow_node_context', 'workflow_node', 'workflow_node_id', 'id');
SELECT create_foreign_key('FK_WORKFLOW_NODE_APPLICATION', 'workflow_node_context', 'application', 'application_id', 'id');
SELECT create_foreign_key('FK_WORKFLOW_NODE_ENVIRONMENT', 'workflow_node_context', 'environment', 'environment_id', 'id');

CREATE TABLE IF NOT EXISTS "workflow_node_hook" (
    id BIGSERIAL PRIMARY KEY,
    uuid TEXT UNIQUE NOT NULL,
    workflow_node_id BIGINT NOT NULL,
    workflow_hook_model_id BIGINT NOT NULL,
    conditions JSONB,
    config JSONB
);

SELECT create_index('workflow_node_hook', 'IDX_WORKFLOW_NODE_HOOK_UUID', 'uuid');
SELECT create_foreign_key('FK_WORKFLOW_NODE_HOOK_WORKFLOW_NODE', 'workflow_node_hook', 'workflow_node', 'workflow_node_id', 'id');
SELECT create_foreign_key('FK_WORKFLOW_NODE_HOOK_WORKFLOW_HOOK_MODEL', 'workflow_node_hook', 'workflow_hook_model', 'workflow_hook_model_id', 'id');

CREATE TABLE IF NOT EXISTS "workflow_hook_model" (
    id BIGSERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    type TEXT NOT NULL,
    image TEXT NOT NULL,
    command TEXT NOT NULL,
    default_config JSONB
);

-- +migrate Down
DROP TABLE workflow_hook_model;
DROP TABLE workflow_node_hook;
DROP TABLE workflow_node_context;
DROP TABLE workflow_node_trigger;
DROP TABLE workflow_node;
DROP TABLE workflow;

