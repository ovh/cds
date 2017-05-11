-- +migrate Up
ALTER TABLE worker ADD COLUMN hatchery_name TEXT NOT NULL DEFAULT '';

-- +migrate Down
ALTER TABLE worker DROP COLUMN hatchery_name;
