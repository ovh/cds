-- +migrate Up
ALTER TABLE workflow_run ADD COLUMN last_sub_num BIGINT NOT NULL DEFAULT 0;
UPDATE workflow_run SET last_sub_num = 0;

ALTER TABLE workflow_run ADD COLUMN last_execution TIMESTAMP WITH TIME ZONE DEFAULT LOCALTIMESTAMP;
UPDATE workflow_run SET last_execution = now();

-- +migrate Down
ALTER TABLE workflow_run DROP COLUMN last_sub_num;
