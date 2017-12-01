-- +migrate Up
ALTER TABLE action ADD COLUMN deprecated BOOLEAN DEFAULT false;

-- +migrate Down
ALTER TABLE action DROP COLUMN deprecated;
