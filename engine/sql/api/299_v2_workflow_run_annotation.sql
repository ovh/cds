-- +migrate Up
ALTER TABLE v2_workflow_run ADD COLUMN "annotations" JSONB;

-- +migrate Down
ALTER TABLE v2_workflow_run DROP COLUMN "annotations";