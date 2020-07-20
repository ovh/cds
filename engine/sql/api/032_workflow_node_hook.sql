-- +migrate Up
ALTER TABLE workflow_hook_model 
    ADD COLUMN author VARCHAR(256) NOT NULL,
    ADD COLUMN description TEXT NOT NULL,
    ADD COLUMN identifier VARCHAR(256) NOT NULL,
    ADD COLUMN icon BYTEA NOT NULL,
    ADD COLUMN disabled BOOLEAN NOT NULL DEFAULT FALSE;

SELECT create_unique_index('workflow_hook_model', 'IDX_WORKFLOW_HOOK_MODEL_NAME', 'name');

-- +migrate Down
ALTER TABLE workflow_hook_model 
    DROP COLUMN author, 
    DROP COLUMN description, 
    DROP COLUMN identifier, 
    DROP COLUMN icon, 
    DROP COLUMN disabled;
