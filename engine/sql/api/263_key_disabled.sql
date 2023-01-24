
-- +migrate Up
ALTER TABLE project_key ADD COLUMN disabled BOOLEAN NOT NULL DEFAULT FALSE;
UPDATE project_key SET disabled = false;

-- +migrate Down
ALTER TABLE project_key DROP COLUMN disabled;
