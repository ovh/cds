-- +migrate Up
ALTER TABLE "worker_model" ADD COLUMN IF NOT EXISTS model_docker JSONB;
ALTER TABLE "worker_model" ADD COLUMN IF NOT EXISTS model_virtual_machine JSONB;
UPDATE worker_model SET model_docker = model WHERE "type" = 'docker';
UPDATE worker_model SET model_virtual_machine = model WHERE "type" != 'docker';
ALTER TABLE worker_capability DROP CONSTRAINT IF EXISTS "fk_worker_capability_worker_model";
SELECT create_foreign_key_idx_cascade('FK_WORKER_CAPABILITY_WORKER_MODEL', 'worker_capability', 'worker_model', 'worker_model_id', 'id');

-- +migrate Down
ALTER TABLE "worker_model" DROP COLUMN model_docker;
ALTER TABLE "worker_model" DROP COLUMN model_virtual_machine;
ALTER TABLE worker_capability DROP CONSTRAINT IF EXISTS "fk_worker_capability_worker_model";
SELECT create_foreign_key('FK_WORKER_CAPABILITY_WORKER_MODEL', 'worker_capability', 'worker_model', 'worker_model_id', 'id');
