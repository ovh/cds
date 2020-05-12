-- +migrate Up
ALTER TABLE "worker_model" DROP COLUMN IF EXISTS owner_id;
ALTER TABLE "worker_model" DROP COLUMN IF EXISTS template;
ALTER TABLE "worker_model" DROP COLUMN IF EXISTS communication;
ALTER TABLE "worker_model" DROP COLUMN IF EXISTS run_script;
ALTER TABLE "worker_model" DROP COLUMN IF EXISTS provision;

ALTER TABLE "worker_model" ADD COLUMN IF NOT EXISTS model_docker JSONB;
ALTER TABLE "worker_model" ADD COLUMN IF NOT EXISTS model_virtual_machine JSONB;
UPDATE worker_model SET model_docker = model WHERE "type" = 'docker';
UPDATE worker_model SET model_virtual_machine = model WHERE "type" != 'docker';
ALTER TABLE "worker_model" DROP COLUMN model;

ALTER TABLE "worker_model" ADD COLUMN IF NOT EXISTS sig BYTEA;
ALTER TABLE "worker_model" ADD COLUMN IF NOT EXISTS signer TEXT;

ALTER TABLE worker_capability DROP CONSTRAINT IF EXISTS "fk_worker_capability_worker_model";
SELECT create_foreign_key_idx_cascade('FK_WORKER_CAPABILITY_WORKER_MODEL', 'worker_capability', 'worker_model', 'worker_model_id', 'id');

CREATE TABLE IF NOT EXISTS "worker_model_secret" (
    id VARCHAR(36) PRIMARY KEY,
    created TIMESTAMP WITH TIME ZONE DEFAULT LOCALTIMESTAMP,
    name VARCHAR(255) NOT NULL,
    worker_model_id BIGINT NOT NULL,
    cipher_value BYTEA,
    sig BYTEA,
    signer TEXT
);
SELECT create_foreign_key_idx_cascade('FK_WORKER_MODEL_SECRET_MODEL', 'worker_model_secret', 'worker_model', 'worker_model_id', 'id');

-- +migrate Down
ALTER table "worker_model" ADD COLUMN IF NOT EXISTS template JSONB;
ALTER table "worker_model" ADD COLUMN IF NOT EXISTS communication TEXT DEFAULT 'http';
ALTER table "worker_model" ADD COLUMN IF NOT EXISTS run_script TEXT DEFAULT '';

ALTER TABLE "worker_model" ADD COLUMN IF NOT EXISTS model JSONB;
UPDATE worker_model SET model = model_docker WHERE "type" = 'docker';
UPDATE worker_model SET model = model_virtual_machine WHERE "type" != 'docker';
ALTER TABLE "worker_model" DROP COLUMN model_docker;
ALTER TABLE "worker_model" DROP COLUMN model_virtual_machine;

ALTER TABLE worker_capability DROP CONSTRAINT IF EXISTS "fk_worker_capability_worker_model";
SELECT create_foreign_key('FK_WORKER_CAPABILITY_WORKER_MODEL', 'worker_capability', 'worker_model', 'worker_model_id', 'id');

DROP TABLE IF EXISTS worker_model_secret;
