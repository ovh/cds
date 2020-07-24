-- +migrate Up
ALTER TABLE pipeline_stage ADD COLUMN conditions JSONB;

-- +migrate Down
ALTER TABLE pipeline_stage DROP COLUMN IF EXISTS conditions;