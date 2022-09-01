-- +migrate Up
SELECT create_index('workflow_run', 'idx_workflow_run_start', 'read_only,start');

-- +migrate Down
DROP INDEX idx_workflow_run_start;
