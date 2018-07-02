-- +migrate Up
ALTER TABLE workflow_node_run_job ADD COLUMN project_id BIGINT DEFAULT 0;
UPDATE workflow_node_run_job SET project_id = 0;

-- +migrate Down
ALTER TABLE workflow_node_run_job DROP COLUMN project_id;