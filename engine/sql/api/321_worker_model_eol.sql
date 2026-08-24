-- +migrate Up
ALTER TABLE worker_model ADD COLUMN eol TIMESTAMP WITH TIME ZONE;

-- +migrate Down
ALTER TABLE worker_model DROP COLUMN eol;
