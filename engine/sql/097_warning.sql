-- +migrate Up
DROP INDEX IDX_WARNING;

select create_unique_index('warning', 'IDX_WARNING', 'project_key, application_name, pipeline_name, environment_name, workflow_name, type, element');

-- +migrate Down