-- +migrate Up
SELECT create_index('item', 'idx_item_status_to_delete', 'status,to_delete');

-- +migrate Down
DROP INDEX idx_item_status_to_delete;
