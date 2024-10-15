-- +migrate Up
CREATE TABLE rbac_region_project
(
    "id"                BIGSERIAL PRIMARY KEY,
    "rbac_id"           uuid    NOT NULL,
    "all_projects"      BOOLEAN NOT NULL DEFAULT FALSE,
    "region_id"         uuid    NOT NULL,
    "role"              VARCHAR(255) NOT NULL,
    sig                 BYTEA,
    signer              TEXT
);
SELECT create_foreign_key_idx_cascade('FK_rbac_region_project', 'rbac_region_project', 'rbac', 'rbac_id', 'id');
SELECT create_foreign_key_idx_cascade('FK_rbac_region_project_region_id', 'rbac_region_project', 'region', 'region_id', 'id');
SELECT create_index('rbac_region_project', 'idx_rbac_region_project_role', 'role');


CREATE TABLE rbac_region_project_keys_project
(
    "id"                        BIGSERIAL PRIMARY KEY,
    "rbac_region_project_id"    BIGINT,
    "project_key"               VARCHAR(255),
    sig                         BYTEA,
    signer                      TEXT
);
SELECT create_foreign_key_idx_cascade('FK_rbac_region_project_projects', 'rbac_region_project_keys_project', 'rbac_region_project', 'rbac_region_project_id', 'id');
SELECT create_foreign_key_idx_cascade('FK_rbac_region_project_keys_project', 'rbac_region_project_keys_project', 'project', 'project_key', 'projectkey');
SELECT create_unique_index('rbac_region_project_keys_project', 'idx_unq_rbac_region_project_keys', 'rbac_region_project_id,project_key');

-- +migrate Down
DROP TABLE rbac_region_project_keys_project;
DROP TABLE rbac_region_project;