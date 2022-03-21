-- +migrate Up
CREATE TABLE rbac
(
    "uuid"          uuid PRIMARY KEY,
    "name"          VARCHAR(255) NOT NULL,
    "created"       TIMESTAMP WITH TIME ZONE DEFAULT LOCALTIMESTAMP,
    "last_modified" TIMESTAMP WITH TIME ZONE DEFAULT LOCALTIMESTAMP,
    sig             BYTEA,
    signer          TEXT
);
SELECT create_unique_index('rbac', 'idx_unq_rbac_name', 'name');

-- GLOBAL PART
CREATE TABLE rbac_global
(
    "id"        BIGSERIAL PRIMARY KEY,
    "rbac_uuid" uuid         NOT NULL,
    "role"      VARCHAR(255) NOT NULL,
    sig         BYTEA,
    signer      TEXT
);
SELECT create_foreign_key_idx_cascade('FK_rbac_global', 'rbac_global', 'rbac', 'rbac_uuid', 'uuid');
SELECT create_index('rbac_global', 'idx_rbac_global_role', 'role');

CREATE TABLE rbac_global_users
(
    "id"             BIGSERIAL PRIMARY KEY,
    "rbac_global_id" BIGINT,
    "user_id"        character varying(36),
    sig              BYTEA,
    signer           TEXT
);
SELECT create_foreign_key_idx_cascade('FK_rbac_global_users', 'rbac_global_users', 'rbac_global', 'rbac_global_id', 'id');
SELECT create_foreign_key_idx_cascade('FK_rbac_global_users_ids', 'rbac_global_users', 'authentified_user', 'user_id', 'id');
SELECT create_unique_index('rbac_global_users', 'idx_unq_rbac_global_users', 'rbac_global_id,user_id');

CREATE TABLE rbac_global_groups
(
    "id"             BIGSERIAL PRIMARY KEY,
    "rbac_global_id" BIGINT,
    "group_id"       BIGINT,
    sig              BYTEA,
    signer           TEXT
);
SELECT create_foreign_key_idx_cascade('FK_rbac_global_groups', 'rbac_global_groups', 'rbac_global', 'rbac_global_id', 'id');
SELECT create_foreign_key_idx_cascade('FK_rbac_global_groups_ids', 'rbac_global_groups', 'group', 'group_id', 'id');
SELECT create_unique_index('rbac_global_groups', 'idx_unq_rbac_global_groups', 'rbac_global_id,group_id');

-- PROJECT
CREATE TABLE rbac_project
(
    "id"        BIGSERIAL PRIMARY KEY,
    "rbac_uuid" uuid         NOT NULL,
    "all"       BOOLEAN      NOT NULL DEFAULT FALSE,
    "role"      VARCHAR(255) NOT NULL,
    sig         BYTEA,
    signer      TEXT
);
SELECT create_foreign_key_idx_cascade('FK_rbac_project', 'rbac_project', 'rbac', 'rbac_uuid', 'uuid');
SELECT create_index('rbac_project', 'idx_rbac_project_role', 'role');

CREATE TABLE rbac_project_projects
(
    "id"              BIGSERIAL PRIMARY KEY,
    "rbac_project_id" BIGINT,
    "project_id"      BIGINT,
    sig               BYTEA,
    signer            TEXT
);
SELECT create_foreign_key_idx_cascade('FK_rbac_project_projects', 'rbac_project_projects', 'rbac_project', 'rbac_project_id', 'id');
SELECT create_foreign_key_idx_cascade('FK_rbac_project_projects_project', 'rbac_project_projects', 'project', 'project_id', 'id');
SELECT create_unique_index('rbac_project_projects', 'idx_unq_rbac_project_projects', 'rbac_project_id,project_id');

CREATE TABLE rbac_project_users
(
    "id"              BIGSERIAL PRIMARY KEY,
    "rbac_project_id" BIGINT,
    "user_id"         character varying(36),
    sig               BYTEA,
    signer            TEXT
);
SELECT create_foreign_key_idx_cascade('FK_rbac_project_users', 'rbac_project_users', 'rbac_project', 'rbac_project_id', 'id');
SELECT create_foreign_key_idx_cascade('FK_rbac_project_users_id', 'rbac_project_users', 'authentified_user', 'user_id', 'id');
SELECT create_unique_index('rbac_project_users', 'idx_unq_rbac_project_users', 'rbac_project_id,user_id');

CREATE TABLE rbac_project_groups
(
    "id"              BIGSERIAL PRIMARY KEY,
    "rbac_project_id" BIGINT,
    "group_id"        BIGINT,
    sig               BYTEA,
    signer            TEXT
);
SELECT create_foreign_key_idx_cascade('FK_rbac_project_groups', 'rbac_project_groups', 'rbac_project', 'rbac_project_id', 'id');
SELECT create_foreign_key_idx_cascade('FK_rbac_project_groups_ids', 'rbac_project_groups', 'group', 'group_id', 'id');
SELECT create_unique_index('rbac_project_groups', 'idx_unq_rbac_project_groups', 'rbac_project_id,group_id');

-- +migrate Down
DROP TABLE rbac_project_groups;
DROP TABLE rbac_project_users;
DROP TABLE rbac_project_projects;
DROP TABLE rbac_project;
DROP TABLE rbac_global_groups;
DROP TABLE rbac_global_users;
DROP TABLE rbac_global;
DROP TABLE rbac;
