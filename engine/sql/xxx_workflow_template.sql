-- +migrate Up

CREATE TABLE IF NOT EXISTS "workflow_templates" (
    id BIGSERIAL PRIMARY KEY,
    group_id BIGINT NOT NULL,
    name TEXT NOT NULL,
    value TEXT,
    pipelines JSONB,
    parameters JSONB
);

select create_index('workflow_templates','IDX_WORKFLOW_TEMPLATES_GROUP_ID', 'group_id');
SELECT create_unique_index('workflow_templates', 'IDX_WORKFLOW_TEMPLATES_NAME', 'group_id,name');

-- +migrate Down

DROP TABLE IF EXISTS workflow_templates;
