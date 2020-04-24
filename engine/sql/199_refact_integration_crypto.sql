-- +migrate Up
ALTER TABLE "project_integration" ADD COLUMN IF NOT EXISTS cipher_config BYTEA;
ALTER TABLE "project_integration" ADD COLUMN IF NOT EXISTS sig BYTEA;
ALTER TABLE "project_integration" ADD COLUMN IF NOT EXISTS signer TEXT;

ALTER TABLE "integration_model" ADD COLUMN IF NOT EXISTS cipher_public_configurations BYTEA;
ALTER TABLE "integration_model" ADD COLUMN IF NOT EXISTS sig BYTEA;
ALTER TABLE "integration_model" ADD COLUMN IF NOT EXISTS signer TEXT;

-- +migrate Down
ALTER TABLE "project_integration" DROP COLUMN cipher_config;
ALTER TABLE "project_integration" DROP COLUMN sig;
ALTER TABLE "project_integration" DROP COLUMN signer;

ALTER TABLE "integration_model" DROP COLUMN cipher_public_configurations;
ALTER TABLE "integration_model" DROP COLUMN sig;
ALTER TABLE "integration_model" DROP COLUMN signer;
