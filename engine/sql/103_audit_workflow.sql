-- +migrate Up
CREATE TABLE workflow_audit (
  id BIGSERIAL PRIMARY KEY,
  project_key VARCHAR(50),
  workflow_id BIGINT,
  triggered_by VARCHAR(100),
  created TIMESTAMP WITH TIME ZONE,
  data_before TEXT,
  data_after TEXT,
  event_type VARCHAR(100),
  data_type VARCHAR(20)
);

-- +migrate Down
DROP TABLE workflow_audit;