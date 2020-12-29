-- +migrate Up
SELECT create_index('item', 'idx_item_type_size', 'type,size');

-- +migrate Down
DROP INDEX "idx_item_type_size";
