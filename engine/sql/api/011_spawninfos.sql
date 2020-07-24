-- +migrate Up
ALTER TABLE pipeline_build_job ADD COLUMN spawninfos JSONB;
UPDATE pipeline_build_job set spawninfos='[]';

-- +migrate Down
ALTER TABLE pipeline_build_job DROP COLUMN spawninfos;
