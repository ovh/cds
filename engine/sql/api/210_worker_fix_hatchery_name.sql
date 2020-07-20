-- +migrate Up

UPDATE "worker" SET hatchery_name = '' WHERE hatchery_name IS NULL;
ALTER TABLE "worker" ALTER COLUMN hatchery_name SET DEFAULT '';

-- +migrate Down

SELECT 1;
