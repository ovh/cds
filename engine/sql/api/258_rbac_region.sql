-- +migrate Up
CREATE TABLE IF NOT EXISTS "rbac_region" (
    id  BIGSERIAL PRIMARY KEY,
    rbac_id uuid NOT NULL,
    region_id uuid NOT NULL,
    role VARCHAR(255) NOT NULL,
    all_users BOOLEAN NOT NULL DEFAULT false,
    sig BYTEA,
    signer TEXT
);
SELECT create_foreign_key_idx_cascade('FK_rbac_region', 'rbac_region', 'rbac', 'rbac_id', 'id');
SELECT create_foreign_key('FK_rbac_region_id', 'rbac_region', 'region', 'region_id', 'id');

CREATE TABLE IF NOT EXISTS "rbac_region_organizations" (
    id  BIGSERIAL PRIMARY KEY,
    rbac_region_id BIGINT NOT NULL,
    organization_id uuid NOT NULL,
    sig BYTEA,
    signer TEXT
);
SELECT create_foreign_key_idx_cascade('FK_rbac_region_organizations', 'rbac_region_organizations', 'rbac_region', 'rbac_region_id', 'id');
SELECT create_unique_index('rbac_region_organizations', 'idx_unq_rbac_region_organizations', 'rbac_region_id,organization_id');

CREATE TABLE IF NOT EXISTS "rbac_region_groups" (
    id  BIGSERIAL PRIMARY KEY,
    rbac_region_id BIGINT,
    group_id BIGINT,
    sig BYTEA,
    signer TEXT
);
SELECT create_foreign_key_idx_cascade('FK_rbac_region_groups', 'rbac_region_groups', 'rbac_region', 'rbac_region_id', 'id');
SELECT create_foreign_key_idx_cascade('FK_rbac_region_groups_ids', 'rbac_region_groups', 'group', 'group_id', 'id');
SELECT create_unique_index('rbac_region_groups', 'idx_unq_rbac_region_groups', 'rbac_region_id,group_id');

CREATE TABLE rbac_region_users
(
    "id"              BIGSERIAL PRIMARY KEY,
    "rbac_region_id" BIGINT,
    "user_id"         character varying(36),
    sig               BYTEA,
    signer            TEXT
);
SELECT create_foreign_key_idx_cascade('FK_rbac_region_users', 'rbac_region_users', 'rbac_region', 'rbac_region_id', 'id');
SELECT create_foreign_key_idx_cascade('FK_rbac_region_users_id', 'rbac_region_users', 'authentified_user', 'user_id', 'id');
SELECT create_unique_index('rbac_region_users', 'idx_unq_rbac_region_users', 'rbac_region_id,user_id');

-- +migrate Down
DROP TABLE rbac_region_users;
DROP TABLE rbac_region_groups;
DROP TABLE rbac_region_organizations;
DROP TABLE rbac_region;


