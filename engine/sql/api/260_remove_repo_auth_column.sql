-- +migrate Up
ALTER TABLE project_repository DROP COLUMN auth;

-- +migrate Down
ALTER TABLE project_repository ADD COLUMN auth BYTEA;

