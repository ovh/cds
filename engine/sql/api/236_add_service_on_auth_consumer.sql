-- +migrate Up
ALTER TABLE "auth_consumer" ADD COLUMN IF NOT EXISTS "service_name" VARCHAR(256);
ALTER TABLE "auth_consumer" ADD COLUMN IF NOT EXISTS "service_type" VARCHAR(256);
ALTER TABLE "auth_consumer" ADD COLUMN IF NOT EXISTS "service_region" VARCHAR(256);

ALTER TABLE "service" ADD COLUMN IF NOT EXISTS "region" VARCHAR(256);

-- +migrate Down
ALTER TABLE "auth_consumer" DROP COLUMN IF EXISTS "service_name";
ALTER TABLE "auth_consumer" DROP COLUMN IF EXISTS "service_type";
ALTER TABLE "auth_consumer" DROP COLUMN IF EXISTS "service_region";

ALTER TABLE "service" DROP COLUMN IF EXISTS "region";
