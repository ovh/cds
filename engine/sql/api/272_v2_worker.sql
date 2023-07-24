-- +migrate Up
CREATE TABLE v2_worker (
  "id"                  uuid PRIMARY KEY,
  "name"                TEXT NOT NULL,
  "last_beat"           TIMESTAMP WITH TIME ZONE DEFAULT LOCALTIMESTAMP,
  "model_name"          TEXT NOT NULL,
  "job_run_id"          uuid NOT NULL,
  "status"               VARCHAR(255),
  "hatchery_id"         uuid NOT NULL,
  "hatchery_name"       TEXT NOT NULL,
  "version"             TEXT NOT NULL,
  "auth_consumer_id"    VARCHAR(36) NOT NULL,
  "os"                  TEXT NOT NULL,
  "arch"                TEXT NOT NULL,
  "cypher_private_key"  BYTEA,
  "sig"                 BYTEA,
  "signer"              TEXT
);
SELECT create_foreign_key_idx_cascade('FK_v2_worker_job', 'v2_worker', 'v2_workflow_run_job', 'job_run_id', 'id');
SELECT create_foreign_key_idx_cascade('FK_v2_worker_hatchery', 'v2_worker', 'hatchery', 'hatchery_id', 'id');
SELECT create_unique_index('v2_worker', 'IDX_v2_worker_name_unq', 'name');



-- +migrate Down
DROP TABLE v2_worker;

