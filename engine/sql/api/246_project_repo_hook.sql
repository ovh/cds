-- +migrate Up
ALTER TABLE project_repository ADD COLUMN hook_configuration JSONB;

-- +migrate Down
ALTER TABLE project_repository DROP COLUMN hook_configuration;
