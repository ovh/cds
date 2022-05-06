-- +migrate Up

ALTER TABLE vcs_project ADD COLUMN IF NOT EXISTS "options" JSONB;

-- +migrate Down

ALTER TABLE vcs_project DROP COLUMN IF EXISTS "options";
