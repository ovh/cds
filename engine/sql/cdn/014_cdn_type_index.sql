-- +migrate Up
SELECT create_index('item', 'idx_item_type_status', 'type,status');
SELECT create_index('storage_unit_item', 'idx_storage_unit_type_item_unit_id', 'type,unit_id');

-- +migrate Down
DROP INDEX "idx_item_type_status";
DROP INDEX "idx_storage_unit_type_item_unit_id";
