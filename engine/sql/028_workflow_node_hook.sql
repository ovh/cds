-- +migrate Up
ALTER TABLE workflow_hook_model 
    ADD COLUMN author TEXT,
    ADD COLUMN description TEXT,
    ADD COLUMN identifier TEXT;

-- +migrate Down
ALTER TABLE workflow_node DROP COLUMN author, description, identifier;
