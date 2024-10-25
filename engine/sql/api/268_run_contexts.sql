-- +migrate Up
ALTER TABLE workflow_node_run ADD COLUMN contexts JSONB;
ALTER TABLE workflow_node_run_job ADD COLUMN contexts JSONB;

-- +migrate Down
ALTER TABLE workflow_node_run DROP COLUMN contexts;
ALTER TABLE workflow_node_run_job DROP COLUMN contexts;

