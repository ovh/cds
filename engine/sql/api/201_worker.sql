-- +migrate Up
ALTER TABLE "worker" ADD COLUMN IF NOT EXISTS cypher_private_key BYTEA;
ALTER TABLE "worker" ADD COLUMN IF NOT EXISTS sig BYTEA;
ALTER TABLE "worker" ADD COLUMN IF NOT EXISTS signer TEXT;


-- +migrate Down
ALTER TABLE "worker" DROP COLUMN cypher_hmac_key;
ALTER TABLE "worker" DROP COLUMN sig;
ALTER TABLE "worker" DROP COLUMN signer;
