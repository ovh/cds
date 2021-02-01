-- +migrate Up
SELECT create_index('item', 'idx_item_status_type', 'status,type');
SELECT create_index('storage_unit_item', 'idx_storage_unit_item_unit_id_type', 'unit_id,type');
DROP INDEX idx_item_status_to_delete;

-- +migrate Down
DROP INDEX idx_item_status_type;
DROP INDEX idx_storage_unit_item_unit_id_type;
SELECT create_index('item', 'idx_item_status_to_delete', 'status,to_delete');
