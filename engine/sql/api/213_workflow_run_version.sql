-- +migrate Up
ALTER TABLE "workflow_run" ADD COLUMN IF NOT EXISTS "version" VARCHAR(256);
SELECT create_unique_index('workflow_run','IDX_WORKFLOW_RUN_WORKFLOW_ID_VERSION','workflow_id,version');

-- +migrate Down
DROP INDEX idx_workflow_run_workflow_id_version;
ALTER TABLE "workflow_run" DROP COLUMN IF EXISTS "version";
