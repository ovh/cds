-- +migrate Up
SELECT create_index('item', 'idx_lastmodified', 'last_modified');

-- +migrate Down
DROP INDEX "idx_lastmodified";
