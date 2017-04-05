-- +migrate Up
ALTER TABLE pipeline_build ADD COLUMN commits JSONB;
UPDATE pipeline_build set commits='[]';

-- +migrate Down
ALTER TABLE pipeline_build DROP COLUMN commits;
