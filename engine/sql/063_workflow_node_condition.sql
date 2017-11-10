-- +migrate Up
ALTER TABLE workflow_node_context ADD COLUMN conditions JSONB;

ALTER TABLE workflow_node_join_trigger DROP COLUMN conditions;
ALTER TABLE workflow_node_join_trigger DROP COLUMN manual;
ALTER TABLE workflow_node_join_trigger DROP COLUMN continue_on_error;

ALTER TABLE workflow_node_trigger DROP COLUMN conditions;
ALTER TABLE workflow_node_trigger DROP COLUMN manual;
ALTER TABLE workflow_node_trigger DROP COLUMN continue_on_error;

-- +migrate Down
ALTER TABLE workflow_node_context DROP COLUMN conditions;

ALTER TABLE workflow_node_join_trigger ADD COLUMN conditions JSONB;
ALTER TABLE workflow_node_join_trigger ADD COLUMN manual boolean;
ALTER TABLE workflow_node_join_trigger ADD COLUMN continue_on_error boolean;

ALTER TABLE workflow_node_trigger ADD COLUMN conditions JSONB;
ALTER TABLE workflow_node_trigger ADD COLUMN manual boolean;
ALTER TABLE workflow_node_trigger ADD COLUMN continue_on_error boolean;