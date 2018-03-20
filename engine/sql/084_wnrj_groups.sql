-- +migrate Up
ALTER TABLE workflow_node_run_job ADD COLUMN exec_groups JSONB;
UPDATE workflow_node_run_job set exec_groups='[]';

-- +migrate Down
ALTER TABLE workflow_node_run_job DROP COLUMN exec_groups;
