
-- +migrate Up

-- remove unused data
ALTER TABLE "action" DROP COLUMN IF EXISTS "data";
UPDATE "action" SET "type" = 'Default' WHERE "type" = '';

ALTER TABLE "action" ALTER COLUMN "name" SET NOT NULL;
ALTER TABLE "action" ALTER COLUMN "type" SET NOT NULL;
ALTER TABLE "action" ALTER COLUMN "description" SET NOT NULL;

-- +migrate Down

SELECT 1;
