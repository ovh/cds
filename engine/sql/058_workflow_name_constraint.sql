-- +migrate Up
SELECT create_unique_index('workflow', 'IDX_WORKFLOW_NAME', 'project_id,name');

-- +migrate Down
DROP INDEX IDX_WORKFLOW_NAME;
