-- +migrate Up
ALTER TABLE workflow_node_trigger ADD COLUMN continue_on_error BOOLEAN DEFAULT false;
ALTER TABLE workflow_node_join_trigger ADD COLUMN continue_on_error BOOLEAN DEFAULT false;

-- +migrate Down
ALTER TABLE workflow_node_trigger DROP COLUMN continue_on_error;
ALTER TABLE workflow_node_join_trigger DROP COLUMN continue_on_error;
