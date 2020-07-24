-- +migrate Up
TRUNCATE TABLE warning;
ALTER TABLE warning ADD COLUMN hash TEXT;
DROP INDEX IDX_WARNING;
select create_unique_index('warning', 'IDX_UNIQ_WARNING', 'hash');

-- +migrate Down
ALTER TABLE warning DROP COLUMN hash;
DROP INDEX IDX_UNIQ_WARNING;
select create_unique_index('warning', 'IDX_WARNING', 'project_key, application_name, pipeline_name, environment_name, workflow_name, type, element');