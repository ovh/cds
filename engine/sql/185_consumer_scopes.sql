-- +migrate Up
ALTER TABLE "auth_consumer" ADD COLUMN scope_details JSONB;

-- +migrate Down
ALTER TABLE "auth_consumer" DROP COLUMN scope_details;
