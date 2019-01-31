-- +migrate Up
ALTER TABLE integration_model RENAME COLUMN file_storage TO storage;
ALTER TABLE integration_model DROP COLUMN block_storage;

-- +migrate Down
ALTER TABLE integration_model RENAME COLUMN storage TO file_storage;
ALTER TABLE integration_model ADD COLUMN block_storage BOOLEAN default false;
