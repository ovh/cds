-- +migrate Up
SELECT create_index('workflow_run', 'IDX_WORKFLOW_LOAD_RUNS_NUM', 'project_id, workflow_id, num');

-- +migrate Down
DROP INDEX IDX_WORKFLOW_LOAD_RUNS_NUM;
