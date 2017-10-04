-- +migrate Up
CREATE TABLE IF NOT EXISTS "pipeline_audit" (
  id BIGSERIAL PRIMARY KEY,
  pipeline_id BIGINT,
  username VARCHAR(100),
  versionned TIMESTAMP WITH TIME ZONE,
  pipeline JSONB,
  action VARCHAR(50)
);
select create_index('pipeline_audit', 'IDX_PIPELINE_AUDIT_ID', 'pipeline_id');

-- +migrate Down
DROP TABLE "pipeline_audit";