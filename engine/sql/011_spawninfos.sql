-- +migrate Up
ALTER TABLE pipeline_build_job ADD COLUMN spawninfos JSONB;

-- +migrate Down
ALTER TABLE pipeline_build_job DROP COLUMN spawninfos;
