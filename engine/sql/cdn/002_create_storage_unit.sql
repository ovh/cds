-- +migrate Up

CREATE TABLE IF NOT EXISTS "storage_unit" (
  id VARCHAR(36) PRIMARY KEY,
  created TIMESTAMP WITH TIME ZONE NOT NULL,
  name VARCHAR(255) NOT NULL,
  config JSONB NOT NULL,
  sig BYTEA,
  signer TEXT
);

CREATE TABLE IF NOT EXISTS "storage_unit_item" (
  id VARCHAR(36) PRIMARY KEY,
  unit_id VARCHAR(36) NOT NULL,
  item_id VARCHAR(36) NOT NULL,
  last_modified TIMESTAMP WITH TIME ZONE NOT NULL,
  cipher_locator BYTEA,
  hash_locator TEXT,
  sig BYTEA,
  signer TEXT,
  to_delete BOOLEAN
);

SELECT create_foreign_key_idx_cascade('FK_storage_unit_item_index', 'storage_unit_item', 'item', 'item_id', 'id');
SELECT create_foreign_key('FK_storage_unit_item_unit', 'storage_unit_item', 'storage_unit', 'unit_id', 'id');
SELECT create_unique_index('storage_unit_item', 'IDX_storage_unit_item_unit_id_item_id', 'unit_id,item_id');
SELECT create_index('storage_unit_item', 'IDX_storage_unit_item_to_delete', 'id,to_delete');
SELECT create_index('storage_unit_item', 'IDX_storage_unit_item_hash_locator', 'id,hash_locator');

-- +migrate Down
DROP TABLE IF EXISTS "storage_unit_item";
DROP TABLE IF EXISTS "storage_unit";
