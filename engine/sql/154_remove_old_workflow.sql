-- +migrate Up
DROP TABLE IF EXISTS pipeline_trigger;
DROP TABLE IF EXISTS pipeline_trigger_parameter;
DROP TABLE IF EXISTS pipeline_trigger_prerequisite;
DROP TABLE IF EXISTS application_pipeline;
DROP TABLE IF EXISTS build_log;
DROP TABLE IF EXISTS artifact;
DROP TABLE IF EXISTS pipeline_build_log;
DROP TABLE IF EXISTS pipeline_build_test;

-- +migrate Down
SELECT 1;