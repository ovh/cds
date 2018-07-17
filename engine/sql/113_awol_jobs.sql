-- +migrate Up
ALTER TABLE workflow_node_run_job ADD COLUMN worker_id TEXT;

-- +migrate Down
ALTER TABLE workflow_node_run_job DROP COLUMN worker_id;


