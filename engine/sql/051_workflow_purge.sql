-- +migrate Up
ALTER TABLE workflow_run ADD COLUMN to_delete BOOLEAN DEFAULT false;
ALTER TABLE workflow ADD COLUMN history_length BIGINT DEFAULT 20;

-- +migrate Down
ALTER TABLE workflow_run DROP COLUMN to_delete;
ALTER TABLE workflow DROP COLUMN history_length;
