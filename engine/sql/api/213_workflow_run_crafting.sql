-- +migrate Up

ALTER TABLE "workflow_run" ADD COLUMN to_craft BOOLEAN;
UPDATE "workflow_run" set to_craft = false;
ALTER TABLE "workflow_run" ALTER COLUMN to_craft SET DEFAULT false;

ALTER TABLE "workflow_run" ADD COLUMN to_craft_opts JSONB;

UPDATE workflow_run SET to_craft = false WHERE to_craft is null;
SELECT create_index('workflow_run', 'IDX_WORKFLOW_RUN_TO_CRAFT', 'id,to_craft');

-- +migrate Down
ALTER TABLE "workflow_run" DROP COLUMN to_craft;
ALTER TABLE "workflow_run" DROP COLUMN to_craft_opts;


