-- +migrate Up
CREATE TABLE v2_workflow_run_result (
  "id"                          uuid  PRIMARY KEY,
  "workflow_run_id"             uuid NOT NULL,
  "workflow_run_job_id"         uuid NOT NULL,
  "issued_at"                   TIMESTAMP WITH TIME ZONE DEFAULT LOCALTIMESTAMP,
  "artifact_manager_metadata"   JSONB,
  "artifact_manager_detail"     JSONB,
  "sync"                        JSONB,
  "status"                      TEXT,
  "type"                        TEXT
);

ALTER TABLE v2_workflow_run_job ADD integrations JSONB;

SELECT create_foreign_key_idx_cascade('FK_v2_workflow_run_result', 'v2_workflow_run_result', 'v2_workflow_run', 'workflow_run_id', 'id');
SELECT create_foreign_key_idx_cascade('FK_v2_workflow_run_job_result', 'v2_workflow_run_result', 'v2_workflow_run_job', 'workflow_run_job_id', 'id');

-- +migrate Down
DROP TABLE v2_workflow_run_result;

ALTER TABLE v2_workflow_run_job DROP integrations;
