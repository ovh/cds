-- +migrate Up

ALTER TABLE workflow_run_result ADD COLUMN IF NOT EXISTS "sync" JSONB;
CREATE INDEX idx_workflow_run_result_sync ON workflow_run_result USING gin (sync);

-- +migrate Down

DROP INDEX idx_workflow_run_result_sync;
ALTER TABLE workflow_run_result DROP COLUMN IF EXISTS "sync";
