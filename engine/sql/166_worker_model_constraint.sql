-- +migrate Up
DROP INDEX IF EXISTS "idx_worker_model_name";
CREATE INDEX idx_worker_model_name ON "worker_model" ("name");

-- +migrate Down
DROP INDEX IF EXISTS "idx_worker_model_name";
SELECT create_unique_index('worker_model','IDX_WORKER_MODEL_NAME','name');
