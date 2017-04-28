-- +migrate Up
CREATE TABLE IF NOT EXISTS "workflow" (
    id BIGSERIAL PRIMARY KEY,
    project_id BIGINT NOT NULL,
    name TEXT NOT NULL,
    root_node_id BIGINT
);

CREATE TABLE IF NOT EXISTS "workflow_node" (
    id BIGSERIAL PRIMARY KEY,
    workflow_id BIGINT NOT NULL,
    pipeline_id BIGINT NOT NULL
);

CREATE TABLE IF NOT EXISTS "workflow_node_trigger" (
    id BIGSERIAL PRIMARY KEY,
    workflow_node_id BIGINT NOT NULL,
    workflow_dest_node_id BIGINT NOT NULL,
    conditions JSONB
);

CREATE TABLE IF NOT EXISTS "workflow_node_context" (
    id BIGSERIAL PRIMARY KEY,
    workflow_node_id BIGINT NOT NULL,
    application_id BIGINT,
    environment_id BIGINT
);

CREATE TABLE IF NOT EXISTS "workflow_node_hook" (
    id BIGSERIAL PRIMARY KEY,
    uuid TEXT UNIQUE NOT NULL,
    workflow_node_id BIGINT NOT NULL,
    workflow_hook_model_id BIGINT NOT NULL,
    conditions JSONB,
    config JSONB
);

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

