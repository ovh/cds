-- +migrate Up
ALTER TABLE pipeline_build_job ADD COLUMN exec_groups JSONB;
UPDATE pipeline_build_job set exec_groups='[]';

-- +migrate Down
ALTER TABLE pipeline_build_job DROP COLUMN exec_groups;
