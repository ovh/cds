-- +migrate Up
ALTER TABLE workflow_node_context ADD COLUMN conditions JSONB;

-- +migrate Down
ALTER TABLE workflow_node_context DROP COLUMN conditions;
