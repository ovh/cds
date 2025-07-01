-- +migrate Up
CREATE TABLE project_run_retention  (
    "id"                uuid PRIMARY KEY,
    "project_key"       VARCHAR(255) NOT NULL,
    "last_modified"     TIMESTAMP WITH TIME ZONE DEFAULT LOCALTIMESTAMP,
    "last_execution"    TIMESTAMP WITH TIME ZONE DEFAULT LOCALTIMESTAMP,
    "last_status"       VARCHAR(255) NOT NULL DEFAULT '',
    "last_report"       JSONB,
    "retention"         JSONB NOT NULL
);
SELECT create_foreign_key_idx_cascade('FK_project_run_retention', 'project_run_retention', 'project', 'project_key', 'projectkey');
SELECT create_unique_index('project_run_retention', 'idx_unq_project_run_retention', 'project_key');


-- +migrate Down
DROP TABLE project_run_retention;
