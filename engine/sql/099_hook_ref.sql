-- +migrate Up
ALTER TABLE workflow_node_hook ADD COLUMN ref VARCHAR(50);
update workflow_node_hook set ref = id;

-- +migrate Down
ALTER TABLE workflow_node_hook DROP COLUMN ref;
