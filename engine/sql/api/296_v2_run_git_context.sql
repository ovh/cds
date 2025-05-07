-- +migrate Up
CREATE INDEX idx_v2_workflow_run_ctx_full_repository 
ON v2_workflow_run (
	( (contexts -> 'git' ->> 'server' ) || '/' || (contexts -> 'git' ->> 'repository') )
);
-- +migrate Down
DROP INDEX IF EXISTS idx_v2_workflow_run_ctx_full_repository;