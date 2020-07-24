-- +migrate Up
TRUNCATE project_key;
TRUNCATE application_key;
TRUNCATE environment_key;

ALTER TABLE project_key DROP COLUMN "private";
ALTER TABLE project_key ADD COLUMN "private" BYTEA;

ALTER TABLE application_key DROP COLUMN "private";
ALTER TABLE application_key ADD COLUMN "private" BYTEA;

ALTER TABLE environment_key DROP COLUMN "private";
ALTER TABLE environment_key ADD COLUMN "private" BYTEA;


ALTER TABLE project_key ADD COLUMN builtin BOOLEAN;

-- +migrate Down
ALTER TABLE project_key DROP COLUMN builtin;
