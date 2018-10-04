-- +migrate Up

CREATE TABLE IF NOT EXISTS "workflow_templates" (
    id BIGSERIAL PRIMARY KEY,
    name TEXT,
    value TEXT,
    pipelines JSONB,
    parameters JSONB
);

-- +migrate Down

DROP TABLE IF EXISTS workflow_templates;
