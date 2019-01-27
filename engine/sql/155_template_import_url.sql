-- +migrate Up

ALTER TABLE workflow_template ADD COLUMN import_url TEXT NOT NULL DEFAULT '';

-- +migrate Down

ALTER TABLE workflow_template DROP COLUMN import_url;
