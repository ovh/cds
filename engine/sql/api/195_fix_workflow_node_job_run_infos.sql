-- +migrate Up
UPDATE workflow_node_run_job SET hatchery_name = '' WHERE hatchery_name is NULL;
UPDATE workflow_node_run_job SET worker_name = '' WHERE worker_name is NULL;

ALTER TABLE workflow_node_run_job ALTER COLUMN hatchery_name SET DEFAULT '';
ALTER TABLE workflow_node_run_job ALTER COLUMN worker_name SET DEFAULT '';

-- +migrate Down
SELECT 1;
