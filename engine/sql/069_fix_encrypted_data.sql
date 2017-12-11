-- +migrate Up
ALTER TABLE encrypted_data ALTER COLUMN content_name SET DATA TYPE VARCHAR(1024);

-- +migrate Down
ALTER TABLE encrypted_data ALTER COLUMN content_name SET DATA TYPE VARCHAR(32);
