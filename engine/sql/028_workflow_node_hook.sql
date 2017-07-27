-- +migrate Up
ALTER TABLE workflow_hook_model 
    ADD COLUMN author VARCHAR(256),
    ADD COLUMN description TEXT,
    ADD COLUMN identifier VARCHAR(256)
    ADD COLUMN icon BYTEA;

SELECT create_unique_index('workflow_hook_model', 'IDX_WORKFLOW_HOOK_MODEL_NAME', 'name');

-- +migrate Down
ALTER TABLE workflow_node DROP COLUMN author, description, identifier, icon;
