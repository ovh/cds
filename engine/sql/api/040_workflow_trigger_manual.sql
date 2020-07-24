-- +migrate Up
ALTER TABLE workflow_node_trigger ADD COLUMN manual boolean;
ALTER TABLE workflow_node_join_trigger ADD COLUMN  manual boolean;
UPDATE workflow_node_trigger SET manual = false;
UPDATE workflow_node_join_trigger SET manual = false;

-- +migrate Down
ALTER TABLE workflow_node_trigger DROP COLUMN manual;
ALTER TABLE workflow_node_join_trigger DROP COLUMN manual;
