-- +migrate Up
ALTER TABLE workflow_run ADD COLUMN status TEXT NOT NULL DEFAULT 'Building';

-- +migrate Down
ALTER TABLE workflow_run DROP COLUMN status;
