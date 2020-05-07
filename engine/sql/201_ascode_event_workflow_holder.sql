-- +migrate Up
ALTER TABLE "as_code_events" ADD COLUMN IF NOT EXISTS workflow_id BIGINT;
SELECT create_foreign_key_idx_cascade('FK_AS_CODE_EVENTS_WORKFLOW', 'as_code_events', 'workflow', 'workflow_id', 'id');

-- +migrate Down
ALTER TABLE "as_code_events" DROP COLUMN workflow_id;

