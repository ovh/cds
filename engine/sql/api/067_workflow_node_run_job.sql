-- +migrate Up
ALTER TABLE workflow_node_run_job DROP COLUMN spawninfos;

-- +migrate Down
ALTER TABLE workflow_node_run_job ADD COLUMN spawninfos JSONB;