-- +migrate Up

CREATE TABLE workflow_outgoing_hook_model
(
    id BIGSERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    type TEXT  NOT NULL,
    command TEXT  NOT NULL,
    default_config JSONB,
    author VARCHAR(256)  NOT NULL,
    description TEXT  NOT NULL,
    identifier VARCHAR(256)  NOT NULL,
    disabled BOOLEAN NOT NULL DEFAULT false,
    icon VARCHAR(50)
);

SELECT create_unique_index('workflow_outgoing_hook_model', 'IDX_WORKFLOW_OUTGOING_HOOK_MODEL_NAME', 'name');

-- +migrate Down

DROP TABLE workflow_outgoing_hook_model;
