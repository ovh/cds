-- +migrate Up
ALTER TABLE workflow_node_run_job ADD COLUMN hatchery_name TEXT;
ALTER TABLE workflow_node_run_job ADD COLUMN worker_name TEXT;

-- +migrate Down
ALTER TABLE workflow_node_run_job DROP COLUMN hatchery_name;
ALTER TABLE workflow_node_run_job DROP COLUMN worker_name;
