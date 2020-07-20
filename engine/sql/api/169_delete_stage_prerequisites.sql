-- +migrate Up
DROP TABLE IF EXISTS pipeline_stage_prerequisite;

-- +migrate Down
CREATE TABLE IF NOT EXISTS "pipeline_stage_prerequisite" (id BIGSERIAL PRIMARY KEY, pipeline_stage_id BIGINT, parameter TEXT, expected_value TEXT);