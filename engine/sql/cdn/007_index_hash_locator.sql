-- +migrate Up
SELECT create_index('storage_unit_item', 'idx_storage_unit_id_hash_locator', 'unit_id,hash_locator');


-- +migrate Down
DROP INDEX idx_storage_unit_id_hash_locator;
