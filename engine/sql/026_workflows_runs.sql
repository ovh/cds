-- +migrate Up
ALTER TABLE workflow_run ADD COLUMN infos JSONB;

-- +migrate Down
ALTER TABLE workflow_run DROP COLUMN infos;
