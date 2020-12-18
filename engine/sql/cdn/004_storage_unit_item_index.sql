-- +migrate Up
SELECT create_index('storage_unit_item', 'idx_storage_unit_item_unit_id', 'unit_id');

-- +migrate Down
DROP INDEX "idx_storage_unit_item_unit_id";
