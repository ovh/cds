-- +migrate Up
ALTER TABLE worker_model
ADD COLUMN last_spawn_err TEXT DEFAULT '',
ADD COLUMN nb_spawn_err INT DEFAULT 0,
ADD COLUMN date_last_spawn_err TIMESTAMP WITH TIME ZONE;

-- +migrate Down
ALTER TABLE worker_model
DROP COLUMN last_spawn_err,
DROP COLUMN nb_spawn_err,
DROP COLUMN date_last_spawn_err;
