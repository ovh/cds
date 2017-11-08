-- +migrate Up
ALTER TABLE workflow_node_run_job ADD COLUMN retry int DEFAULT 0;
ALTER TABLE worker ADD COLUMN job_type text;
-- +migrate Down
ALTER TABLE workflow_node_run_job DROP COLUMN retry;
ALTER TABLE worker DROP COLUMN job_type;
