-- +migrate Up
ALTER TABLE project ADD COLUMN metadata JSONB;

-- +migrate Down
ALTER table project DROP COLUMN metadata;