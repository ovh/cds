-- +migrate Up

CREATE TABLE rbac_project_keys
(
    "id"              BIGSERIAL PRIMARY KEY,
    "rbac_project_id" BIGINT,
    "project_key"     VARCHAR(256),
    sig               BYTEA,
    signer            TEXT
);
SELECT create_foreign_key_idx_cascade('FK_rbac_project_keys', 'rbac_project_keys', 'rbac_project', 'rbac_project_id', 'id');
SELECT create_foreign_key_idx_cascade('FK_rbac_project_keys_project', 'rbac_project_keys', 'project', 'project_key', 'projectkey');
SELECT create_unique_index('rbac_project_keys', 'idx_unq_rbac_project_keys', 'rbac_project_id,project_key');

DROP TABLE rbac_project_projects;

-- +migrate Down

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

DROP TABLE rbac_project_keys;