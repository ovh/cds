-- +migrate Up
ALTER TABLE "auth_consumer" ADD COLUMN validity_periods JSONB;

-- +migrate Down
ALTER TABLE "auth_consumer" DROP COLUMN validity_periods JSONB;
