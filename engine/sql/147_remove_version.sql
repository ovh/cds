-- +migrate Up

ALTER TABLE workflow_run DROP COLUMN version;

-- +migrate Down

ALTER TABLE workflow_run ADD COLUMN version INT;
UPDATE workflow_run SET version = 2;
ALTER TABLE workflow_run ALTER COLUMN version SET DEFAULT 2;
UPDATE workflow_run SET version = 2;
