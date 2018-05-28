-- +migrate Up
ALTER TABLE artifact ADD COLUMN sha512sum TEXT;
ALTER TABLE workflow_node_run_artifacts ADD COLUMN sha512sum TEXT;

-- +migrate Down
ALTER TABLE artifact DROP COLUMN sha512sum;
ALTER TABLE workflow_node_run_artifacts DROP COLUMN sha512sum;
