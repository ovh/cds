-- +migrate Up
ALTER TABLE storage_unit_item ADD COLUMN IF NOT EXISTS type VARCHAR(64);

UPDATE storage_unit_item SET type = item.type FROM item WHERE item.id = storage_unit_item.item_id AND (storage_unit_item.type IS NULL OR storage_unit_item.type = '');


-- +migrate Down
ALTER TABLE  storage_unit_item DROP COLUMN type;
