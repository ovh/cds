-- +migrate Up
ALTER TABLE worker ADD COLUMN model_name TEXT DEFAULT '';

-- +migrate Down
ALTER TABLE worker DROP COLUMN model_name;

