-- +migrate Up
ALTER TABLE worker_model ADD COLUMN is_deprecated BOOLEAN DEFAULT FALSE;

-- +migrate Down
ALTER TABLE worker_model DROP COLUMN is_deprecated;
