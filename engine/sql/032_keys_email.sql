-- +migrate Up
ALTER TABLE project_key ADD COLUMN key_id TEXT;
ALTER TABLE application_key ADD COLUMN key_id TEXT;
ALTER TABLE environment_key ADD COLUMN key_id TEXT;

-- +migrate Down
ALTER TABLE project_key DROP COLUMN key_id;
ALTER TABLE application_key DROP COLUMN key_id;
ALTER TABLE environment_key DROP COLUMN key_id;

