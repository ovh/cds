-- +migrate Up

ALTER TABLE "worker" ADD COLUMN IF NOT EXISTS hatchery_name VARCHAR(256);
ALTER TABLE "worker" ALTER COLUMN hatchery_id DROP NOT NULL;

-- +migrate Down

ALTER TABLE "worker" DROP COLUMN IF EXISTS hatchery_name;
ALTER TABLE "worker" ALTER COLUMN hatchery_id SET NOT NULL;
