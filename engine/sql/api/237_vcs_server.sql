-- +migrate Up
CREATE TABLE vcs_server
(
    "id"            uuid PRIMARY KEY,
    "name"          VARCHAR(255) NOT NULL,
    "type"          VARCHAR(64) NOT NULL,    
    "project_id"    BIGINT,
    "description"   VARCHAR(255) NOT NULL,
    "url"           VARCHAR(255) NOT NULL,
    "auth_token"    BYTEA,
    "auth_username" VARCHAR(255) NOT NULL,
    "created"       TIMESTAMP WITH TIME ZONE DEFAULT LOCALTIMESTAMP,    
    "last_modified" TIMESTAMP WITH TIME ZONE DEFAULT LOCALTIMESTAMP,
    "created_by"    VARCHAR(255) NOT NULL,
    "sig"           BYTEA,
    "signer"        TEXT
);
SELECT create_unique_index('vcs_server', 'idx_unq_name_project_id', 'name,project_id');
SELECT create_foreign_key_idx_cascade('fk_vcs_server_project', 'vcs_server', 'project', 'project_id', 'id');

-- +migrate Down
DROP TABLE vcs_server;
