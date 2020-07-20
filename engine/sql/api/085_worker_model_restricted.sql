-- +migrate Up
ALTER TABLE worker_model ADD COLUMN restricted BOOLEAN DEFAULT FALSE;

-- +migrate Down
ALTER TABLE worker_model DROP COLUMN restricted;
