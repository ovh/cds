-- +migrate Up

CREATE TABLE workflow_outgoing_hook_model
(
    id BIGSERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    type TEXT  NOT NULL,
    command TEXT  NOT NULL,
    default_config JSONB,
    author VARCHAR(256)  NOT NULL,
    description TEXT  NOT NULL,
    identifier VARCHAR(256)  NOT NULL,
    disabled BOOLEAN NOT NULL DEFAULT false,
    icon VARCHAR(50)
);

SELECT create_unique_index('workflow_outgoing_hook_model', 'IDX_WORKFLOW_OUTGOING_HOOK_MODEL_NAME', 'name');

CREATE TABLE workflow_node_outgoing_hook
(
    id BIGSERIAL PRIMARY KEY,
    workflow_node_id BIGINT NOT NULL,
    workflow_hook_model_id BIGINT NOT NULL,
    config JSONB
);

SELECT create_foreign_key_idx_cascade('FK_WORKFLOW_NODE_OUTGOING_HOOK_WORKFLOW_NODE', 'workflow_node_outgoing_hook', 'workflow_node', 'workflow_node_id', 'id');
SELECT create_foreign_key('FK_WORKFLOW_NODE_HOOK_WORKFLOW_OUTGOING_HOOK_MODEL', 'workflow_node_outgoing_hook', 'workflow_hook_model', 'workflow_hook_model_id', 'id');

CREATE TABLE workflow_node_outgoing_hook_trigger
(
    id BIGSERIAL PRIMARY KEY,
    workflow_node_outgoing_hook_id BIGINT NOT NULL,
    workflow_dest_node_id BIGINT NOT NULL
);

SELECT create_foreign_key_idx_cascade('FK_WORKFLOW_NODE_OUTGOING_HOOK_TRIGGER_WORKFLOW_NODE_OUTGOING_HOOK', 'workflow_node_outgoing_hook_trigger', 'workflow_node_outgoing_hook', 'workflow_node_outgoing_hook_id', 'id');
SELECT create_foreign_key('FK_WORKFLOW_NODE_OUTGOING_HOOK__TRIGGER_WORKFLOW_NODE_DEST', 'workflow_node_outgoing_hook_trigger', 'workflow_node', 'workflow_dest_node_id', 'id');

ALTER TABLE workflow_node ADD COLUMN workflow_outgoing_hook_trigger_id BIGINT;
ALTER TABLE workflow_node ADD CONSTRAINT FK_WORKFLOW_NODE_TRIGGER_OUTGOING_HOOK FOREIGN KEY(workflow_outgoing_hook_trigger_id) REFERENCES workflow_node_outgoing_hook_trigger(id) ON DELETE CASCADE DEFERRABLE INITIALLY DEFERRED;

-- +migrate Down

DROP TABLE workflow_outgoing_hook_model CASCADE;
DROP TABLE workflow_node_outgoing_hook CASCADE;
DROP TABLE workflow_node_outgoing_hook_trigger CASCADE;
ALTER TABLE workflow_node DROP CONSTRAINT FK_WORKFLOW_NODE_TRIGGER_OUTGOING_HOOK;
ALTER TABLE workflow_node DROP COLUMN workflow_outgoing_hook_trigger_id;

