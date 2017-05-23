-- +migrate Up
ALTER TABLE worker_model ADD COLUMN user_last_modified TIMESTAMP WITH TIME ZONE DEFAULT LOCALTIMESTAMP;

-- +migrate Down
ALTER TABLE worker_model DROP COLUMN user_last_modified;
