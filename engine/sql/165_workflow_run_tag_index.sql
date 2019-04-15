-- +migrate Up
SELECT create_index('workflow_run_tag', 'IDX_WORKFLOW_RUN_TAG_VALUE', 'tag,value');

-- +migrate Down
DROP INDEX idx_workflow_run_tag_value;