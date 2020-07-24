-- +migrate Up
ALTER TABLE workflow_run ADD COLUMN status TEXT NOT NULL DEFAULT 'Waiting';
UPDATE workflow_run SET status = 'Success';

-- +migrate Down
ALTER TABLE workflow_run DROP COLUMN status;
