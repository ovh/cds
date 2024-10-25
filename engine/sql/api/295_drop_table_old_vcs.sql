-- +migrate Up
DROP TABLE project_vcs_server_link_data;
DROP TABLE project_vcs_server_link;


-- +migrate Down

CREATE TABLE IF NOT EXISTS "project_vcs_server_link" (
    "id" BIGSERIAL PRIMARY KEY,
    "project_id" BIGINT NOT NULL,
    "name" TEXT NOT NULL,
    "username" TEXT,
    "vcs_project" TEXT,
    "sig" BYTEA,
    "signer" TEXT
);

SELECT create_unique_index('project_vcs_server_link','IDX_PROJECT_VCS_SERVER_LINK_NAME','project_id,name');
SELECT create_foreign_key_idx_cascade('FK_DX_PROJECT_VCS_SERVER_LINK_PROJECT', 'project_vcs_server_link', 'project', 'project_id', 'id');

CREATE TABLE IF NOT EXISTS "project_vcs_server_link_data" (
    "id" BIGSERIAL PRIMARY KEY,
    "project_vcs_server_link_id" BIGINT NOT NULL,
    "key" TEXT NOT NULL,
    "cipher_value" BYTEA,
    "sig" BYTEA,
    "signer" TEXT
);

SELECT create_unique_index('project_vcs_server_link_data','IDX_PROJECT_VCS_SERVER_LINK_DATA_KEY','project_vcs_server_link_id,key');
SELECT create_foreign_key_idx_cascade('FK_DX_PROJECT_VCS_SERVER_LINK_DATA_PROJECT', 'project_vcs_server_link_data', 'project_vcs_server_link', 'project_vcs_server_link_id', 'id');
