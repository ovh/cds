-- +migrate Up
ALTER TABLE "auth_consumer" ADD COLUMN validity_periods JSONB;
ALTER TABLE "auth_consumer" ADD COLUMN last_authentication TIMESTAMP WITH TIME ZONE;

-- +migrate Down
ALTER TABLE "auth_consumer" DROP COLUMN validity_periods;
ALTER TABLE "auth_consumer" DROP COLUMN last_authentication;
