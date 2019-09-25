-- +migrate Up
ALTER TABLE "workflow_as_code_events" ADD COLUMN data JSONB;
ALTER TABLE "workflow_as_code_events" ADD COLUMN from_repository VARCHAR(300);
ALTER TABLE "workflow_as_code_events" DROP COLUMN workflow_id;
ALTER TABLE "workflow_as_code_events" RENAME TO "as_code_events";
CREATE INDEX idx_as_code_events_data ON as_code_events USING gin (data);

-- +migrate Down
ALTER TABLE "as_code_events" RENAME TO "workflow_as_code_events";
ALTER TABLE "workflow_as_code_events" DROP COLUMN data;
ALTER TABLE "workflow_as_code_events" DROP COLUMN from_repository;
ALTER TABLE "workflow_as_code_events" ADD COLUMN workflow_id BIGINT;
SELECT create_foreign_key_idx_cascade('FK_AS_CODE_EVENT', 'workflow_as_code_events', 'workflow', 'workflow_id', 'id');
