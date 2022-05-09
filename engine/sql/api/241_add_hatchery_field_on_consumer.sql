-- +migrate Up
ALTER TABLE "auth_consumer" ADD COLUMN IF NOT EXISTS "service_ignore_job_with_no_region" BOOLEAN;

ALTER TABLE "service" ADD COLUMN IF NOT EXISTS "ignore_job_with_no_region" BOOLEAN;

-- +migrate Down
ALTER TABLE "auth_consumer" DROP COLUMN IF EXISTS "service_ignore_job_with_no_region";

ALTER TABLE "service" DROP COLUMN IF EXISTS "ignore_job_with_no_region";
