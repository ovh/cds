-- +migrate Up
CREATE TABLE project_run_retention  (
    "id"                uuid PRIMARY KEY,
    "project_key"       VARCHAR(255) NOT NULL,
    "last_modified"     TIMESTAMP WITH TIME ZONE DEFAULT LOCALTIMESTAMP,
    "retention"         JSONB NOT NULL
);
SELECT create_foreign_key_idx_cascade('FK_project_run_retention', 'project_run_retention', 'project', 'project_key', 'projectkey');
SELECT create_unique_index('project_run_retention', 'idx_unq_project_run_retention', 'project_key');

ALTER TABLE project DROP COLUMN workflow_retention;
ALTER TABLE v2_workflow_run DROP COLUMN retention_date;

-- +migrate Down
DROP TABLE project_run_retention;

ALTER TABLE project ADD COLUMN workflow_retention integer default '90';
ALTER TABLE v2_workflow_run ADD COLUMN retention_date TIMESTAMP WITH TIME ZONE DEFAULT LOCALTIMESTAMP;
UPDATE v2_workflow_run SET retention_date = started + interval '90' day;
SELECT create_index('v2_workflow_run','IDX_V2_WORKFLOW_RUN_RETENTION_DATE','retention_date');

