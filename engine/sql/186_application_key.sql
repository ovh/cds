-- +migrate Up
ALTER TABLE "application_key" ADD COLUMN IF NOT EXISTS sig BYTEA;
ALTER TABLE "application_key" ADD COLUMN IF NOT EXISTS signer TEXT;

-- +migrate Down

ALTER TABLE "application_key" DROP COLUMN sig;
ALTER TABLE "application_key" DROP COLUMN signer;
