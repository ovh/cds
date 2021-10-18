-- +migrate Up
CREATE TABLE "worker_hook" (
    id BIGSERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    config JSONB NOT NULL
);

SELECT create_unique_index('worker_hook','IDX_WORKER_HOOK_NAME','name');

-- +migrate Down
DROP TABLE "worker_hook";
