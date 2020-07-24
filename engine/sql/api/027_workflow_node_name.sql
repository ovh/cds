-- +migrate Up
ALTER TABLE workflow_node ADD COLUMN name TEXT;

-- +migrate Down
ALTER TABLE workflow_node DROP COLUMN name;
