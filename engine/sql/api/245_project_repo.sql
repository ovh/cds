-- +migrate Up
CREATE TABLE project_repository
(
    "id"                uuid PRIMARY KEY,
    "name"              VARCHAR(255) NOT NULL,
    "vcs_project_id"    uuid,
    "created"           TIMESTAMP WITH TIME ZONE DEFAULT LOCALTIMESTAMP,
    "created_by"        VARCHAR(255) NOT NULL,
    "sig"           BYTEA,
    "signer"        TEXT
);
SELECT create_unique_index('project_repository', 'idx_unq_project_repository', 'vcs_project_id,name');
SELECT create_foreign_key_idx_cascade('fk_project_repository_vcs_project', 'project_repository', 'vcs_project', 'vcs_project_id', 'id');

-- +migrate Down
DROP TABLE project_repository;
