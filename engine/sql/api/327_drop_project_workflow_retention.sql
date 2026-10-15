-- +migrate Up
ALTER TABLE project DROP COLUMN IF EXISTS workflow_retention;

-- +migrate Down
ALTER TABLE project ADD COLUMN IF NOT EXISTS workflow_retention integer default '90';
