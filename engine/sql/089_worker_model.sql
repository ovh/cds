-- +migrate Up
ALTER TABLE worker_model ADD COLUMN model JSONB;

-- +migrate Down
ALTER TABLE worker_model DROP COLUMN model;
