-- +migrate Up
ALTER TABLE workflow ADD COLUMN to_delete BOOLEAN DEFAULT FALSE;

-- +migrate Down
ALTER TABLE workflow DROP COLUMN to_delete;