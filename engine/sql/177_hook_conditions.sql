-- +migrate Up
ALTER TABLE w_node_hook ADD COLUMN conditions JSONB;

-- +migrate Down
ALTER TABLE w_node_hook DROP COLUMN conditions;
