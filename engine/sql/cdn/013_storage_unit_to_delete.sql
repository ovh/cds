-- +migrate Up
ALTER TABLE storage_unit ADD COLUMN IF NOT EXISTS to_delete BOOLEAN;

-- +migrate Down
ALTER TABLE  storage_unit DROP COLUMN to_delete;
