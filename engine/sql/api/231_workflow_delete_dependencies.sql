-- +migrate Up
ALTER TABLE "workflow" ADD COLUMN IF NOT EXISTS "to_delete_with_dependencies" BOOLEAN;

-- +migrate Down
ALTER TABLE "workflow" DROP COLUMN IF EXISTS "to_delete_with_dependencies"
