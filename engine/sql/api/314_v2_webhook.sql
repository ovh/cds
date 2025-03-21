-- +migrate Up
CREATE TABLE project_webhook (
    "id"                    uuid PRIMARY KEY,
    "project_key"           VARCHAR(255) NOT NULL,
    "vcs_server"            VARCHAR(255) NOT NULL,
    "repository"            VARCHAR(255) NOT NULL,
    "workflow"              VARCHAR(255) NOT NULL DEFAULT '',
    "type"                  VARCHAR(255) NOT NULL,
    "created"               TIMESTAMP WITH TIME ZONE DEFAULT LOCALTIMESTAMP,
    "username"              TEXT NOT NULL
);
SELECT create_foreign_key_idx_cascade('FK_project_webhook', 'project_webhook', 'project', 'project_key', 'projectkey');

-- +migrate Down
DROP TABLE project_webhook;