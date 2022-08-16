-- +migrate Up

ALTER TABLE workflow_run_result ADD COLUMN IF NOT EXISTS "sync" JSONB;

-- +migrate Down

ALTER TABLE workflow_run_result DROP COLUMN IF EXISTS "sync";
