-- +migrate Up
ALTER TABLE test_encrypted_data ADD COLUMN sensitive_json_data BYTEA;

-- +migrate Down
ALTER TABLE test_encrypted_data DROP COLUMN sensitive_json_data;
