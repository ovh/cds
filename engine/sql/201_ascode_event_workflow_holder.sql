-- +migrate Up
ALTER TABLE "as_code_events" ADD COLUMN IF NOT EXISTS workflow_id BYTEA;

-- +migrate Down
ALTER TABLE "as_code_events" DROP COLUMN workflow_id;

