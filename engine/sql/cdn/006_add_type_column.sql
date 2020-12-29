-- +migrate Up
ALTER TABLE storage_unit_item ADD COLUMN type VARCHAR(64);

-- +migrate Down
ALTER TABLE  storage_unit_item DROP COLUMN type;
