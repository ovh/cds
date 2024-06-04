-- +migrate Up
ALTER TABLE project ADD COLUMN workflow_retention integer default '90';
ALTER TABLE v2_workflow_run ADD COLUMN retention_date TIMESTAMP WITH TIME ZONE DEFAULT LOCALTIMESTAMP;
UPDATE v2_workflow_run SET retention_date = started + interval '90' day;
SELECT create_index('v2_workflow_run','IDX_V2_WORKFLOW_RUN_RETENTION_DATE','retention_date');

-- +migrate Down
ALTER TABLE project DROP COLUMN workflow_retention;
ALTER TABLE v2_workflow_run DROP COLUMN retention;