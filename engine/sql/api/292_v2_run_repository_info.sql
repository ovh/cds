-- +migrate Up
ALTER TABLE v2_workflow_run ADD COLUMN vcs_server VARCHAR(255) NOT NULL DEFAULT '';
ALTER TABLE v2_workflow_run ADD COLUMN repository VARCHAR(255) NOT NULL DEFAULT '';
SELECT create_index('v2_workflow_run', 'idx_v2_workflow_run_username', 'username');
SELECT create_index('v2_workflow_run', 'idx_v2_workflow_run_workflow_name', 'workflow_name');
CREATE INDEX idx_v2_workflow_run_git_ref ON v2_workflow_run((contexts -> 'git' ->> 'ref'));

-- +migrate Down
ALTER TABLE v2_workflow_run DROP COLUMN vcs_server;
ALTER TABLE v2_workflow_run DROP COLUMN repository;
DROP INDEX IF EXISTS idx_v2_workflow_run_username;
DROP INDEX IF EXISTS idx_v2_workflow_run_workflow_name;
DROP INDEX IF EXISTS idx_v2_workflow_run_git_ref;