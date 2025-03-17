-- +migrate Up
CREATE TABLE project_repository_hook (
    "id"                    uuid PRIMARY KEY,
    "project_key"           VARCHAR(255) NOT NULL,
    "vcs_server"            VARCHAR(255) NOT NULL,
    "repository"            VARCHAR(255) NOT NULL,
    "created"               TIMESTAMP WITH TIME ZONE DEFAULT LOCALTIMESTAMP,
    "username"              TEXT NOT NULL
);
SELECT create_foreign_key_idx_cascade('FK_project_repository_hook', 'project_repository_hook', 'project', 'project_key', 'projectkey');

-- +migrate Down
DROP TABLE project_repository_hook;