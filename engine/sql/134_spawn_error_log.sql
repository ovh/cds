-- +migrate Up

ALTER TABLE worker_model ADD COLUMN last_spawn_err_log TEXT;

-- +migrate Down

ALTER TABLE worker_model DROP COLUMN last_spawn_err_log;