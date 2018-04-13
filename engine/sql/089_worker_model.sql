-- +migrate Up
ALTER TABLE worker_model ADD COLUMN model JSONB;
ALTER TABLE worker_model ADD COLUMN registered_os text;
ALTER TABLE worker_model ADD COLUMN registered_arch text;

CREATE TABLE IF NOT EXISTS worker_model_pattern (id BIGSERIAL PRIMARY KEY, name text, type text, model JSONB);
select create_unique_index('worker_model_pattern', 'IDX_WORKER_MODEL_PATTERN_NAME_TYPE', 'name,type');

-- +migrate Down
ALTER TABLE worker_model DROP COLUMN model;
ALTER TABLE worker_model DROP COLUMN registered_os;
ALTER TABLE worker_model DROP COLUMN registered_arch;
DROP TABLE worker_model_pattern;
