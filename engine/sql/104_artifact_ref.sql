-- +migrate Up
ALTER TABLE workflow_node_run_artifacts ADD COLUMN ref TEXT;

-- +migrate Down
ALTER TABLE workflow_node_run_artifacts DROP COLUMN ref;
