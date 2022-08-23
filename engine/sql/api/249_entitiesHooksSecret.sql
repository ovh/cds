-- +migrate Up
ALTER TABLE project_repository ADD COLUMN hook_sign_key BYTEA;

-- +migrate Down
ALTER TABLE project_repository DROP COLUMN hook_sign_key;
