
-- +migrate Up
ALTER TABLE entity ADD COLUMN file_path TEXT NOT NULL;

-- +migrate Down
ALTER TABLE entity DROP COLUMN file_path;
