-- +migrate Up

ALTER TABLE "storage_unit_index" ADD COLUMN IF NOT EXISTS "created" TIMESTAMP WITH TIME ZONE NOT NULL;
ALTER TABLE "storage_unit_index" ADD COLUMN IF NOT EXISTS "complete" BOOLEAN;

-- +migrate Down

ALTER TABLE "storage_unit_index" DROP COLUMN IF EXISTS "created";
ALTER TABLE "storage_unit_index" DROP COLUMN IF EXISTS "complete";
