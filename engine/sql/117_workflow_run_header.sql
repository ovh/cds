-- +migrate Up
ALTER TABLE workflow_run ADD COLUMN header JSONB;
ALTER TABLE workflow_node_run ADD COLUMN header JSONB;
ALTER TABLE workflow_node_run_job ADD COLUMN header JSONB;

-- +migrate Down
ALTER TABLE workflow_run DROP COLUMN header;
ALTER TABLE workflow_node_run DROP COLUMN header;
ALTER TABLE workflow_node_run_job DROP COLUMN header;