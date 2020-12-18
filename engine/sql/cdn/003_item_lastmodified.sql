-- +migrate Up
SELECT create_index('item', 'idx_item_created', 'created');

-- +migrate Down
DROP INDEX "idx_item_created";
