-- +migrate Up
ALTER TABLE workflow_node_run_job DROP COLUMN spawn_attempts;

-- +migrate Down
ALTER TABLE workflow_node_run_job ADD COLUMN spawn_attempts bigint[];
