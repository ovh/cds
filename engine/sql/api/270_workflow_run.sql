-- +migrate Up
CREATE TABLE v2_workflow_run (
  "id"              uuid PRIMARY KEY,
  "project_key"     VARCHAR(255)  NOT NULL,
  "vcs_server_id"   uuid          NOT NULL,
  "repository_id"   uuid          NOT NULL,
  "workflow_name"   VARCHAR(255)  NOT NULL,
  "workflow_sha"    VARCHAR(255)  NOT NULL,
  "workflow_ref"    VARCHAR(512)  NOT NULL,
  "status"          VARCHAR(100)  NOT NULL,
  "run_number"      BIGINT        NOT NULL,
  "run_attempt"     BIGINT        NOT NULL,
  "started"         TIMESTAMP WITH TIME ZONE DEFAULT LOCALTIMESTAMP,
  "last_modified"   TIMESTAMP WITH TIME ZONE DEFAULT LOCALTIMESTAMP,
  "to_delete"       BOOLEAN       DEFAULT false,
  "workflow_data"   TEXT          NOT NULL,
  "contexts"        TEXT          NOT NULL,
  "user_id"         VARCHAR(36)   NOT NULL,
  "event"           JSONB         NOT NULL,
  "sig"             BYTEA,
  "signer"          TEXT
);
SELECT create_foreign_key_idx_cascade('FK_v2_workflow_run_project', 'v2_workflow_run', 'project', 'project_key', 'projectkey');
SELECT create_index('v2_workflow_run', 'idx_v2_workflow_run_status', 'status');
SELECT create_foreign_key('FK_v2_workflow_run_user', 'v2_workflow_run', 'authentified_user', 'user_id', 'id');

CREATE TABLE v2_workflow_run_job (
  "id"              uuid PRIMARY KEY,
  "workflow_run_id" uuid NOT NULL,
  "status"          VARCHAR(255),
  "queued"          TIMESTAMP WITH TIME ZONE DEFAULT LOCALTIMESTAMP,
  "started"         TIMESTAMP WITH TIME ZONE DEFAULT LOCALTIMESTAMP,
  "ended"           TIMESTAMP WITH TIME ZONE DEFAULT LOCALTIMESTAMP,
  "job_id"          VARCHAR(255),
  "job"             TEXT  NOT NULL,
  "outputs"         TEXT,
  "worker_id"       VARCHAR(36),
  "worker_name"     VARCHAR(255),
  "hatchery_name"   VARCHAR(512),
  "sig"             BYTEA,
  "signer"          TEXT
);
SELECT create_foreign_key_idx_cascade('FK_v2_workflow_run_job', 'v2_workflow_run_job', 'v2_workflow_run', 'workflow_run_id', 'id');
SELECT create_index('v2_workflow_run_job', 'idx_v2_workflow_run_job_status', 'status');

CREATE TABLE v2_workflow_run_info (
  "id"              uuid PRIMARY KEY,
  "workflow_run_id" uuid NOT NULL,
  "issued_at"       TIMESTAMP WITH TIME ZONE DEFAULT LOCALTIMESTAMP,
  "level"            VARCHAR(50),
  "message"         TEXT
);
SELECT create_foreign_key_idx_cascade('FK_v2_workflow_run_infos', 'v2_workflow_run_info', 'v2_workflow_run', 'workflow_run_id', 'id');

-- +migrate Down
DROP TABLE v2_workflow_run_info;
DROP TABLE v2_workflow_run_job;
DROP TABLE v2_workflow_run;

