-- +migrate Up
SELECT create_index('v2_workflow_run', 'idx_v2_workflow_run_vcs_server', 'vcs_server');
SELECT create_index('v2_workflow_run', 'idx_v2_workflow_run_repository', 'repository');

-- +migrate Down
DROP INDEX IF EXISTS idx_v2_workflow_run_vcs_server;
DROP INDEX IF EXISTS idx_v2_workflow_run_repository;