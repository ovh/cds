-- +migrate Up
CREATE TABLE IF NOT EXISTS "pipeline_build_log" (
  id BIGSERIAL PRIMARY KEY,
  pipeline_build_job_id BIGINT,
  pipeline_build_id BIGINT,
  start TIMESTAMP WITH TIME ZONE,
  last_modified TIMESTAMP WITH TIME ZONE,
  done TIMESTAMP WITH TIME ZONE,
  step_order TEXT,
  "value" BYTEA
);
select create_foreign_key('FK_BUILD_LOG_PIPELINE_BUILD', 'pipeline_build_log', 'pipeline_build', 'pipeline_build_id', 'id');
select create_unique_index('pipeline_build_log', 'IDX_PIPELINE_BUILD_LOG_UNIQUE', 'pipeline_build_id,step_order');


-- +migrate Down
DROP TABLE pipeline_build_log;