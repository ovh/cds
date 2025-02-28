-- +migrate Up
CREATE TABLE project_concurrency
(
    "id"                    uuid PRIMARY KEY,
    "project_key"           VARCHAR(255) NOT NULL,
    "name"                  VARCHAR(255) NOT NULL,
    "description"           TEXT NOT NULL,
    "order"                 VARCHAR(255) NOT NULL,
    "pool"                  BIGINT NOT NULL DEFAULT 1,
    "cancel_in_progress"    BOOLEAN NOT NULL DEFAULT false,
    "last_modified"         TIMESTAMP WITH TIME ZONE DEFAULT LOCALTIMESTAMP
);
SELECT create_foreign_key_idx_cascade('FK_v2_project_concurrency_project', 'project_concurrency', 'project', 'project_key', 'projectkey');
SELECT create_unique_index('project_concurrency', 'idx_unq_project_concurrency', 'project_key,name');

CREATE INDEX IDX_V2_WORKFLOW_RUN_JOB_CONCURRENCY ON v2_workflow_run_job(project_key, (concurrency->>'name'), (concurrency->>'scope'));

-- +migrate Down
DROP TABLE project_concurrency;
DROP INDEX IDX_V2_WORKFLOW_RUN_JOB_CONCURRENCY;
