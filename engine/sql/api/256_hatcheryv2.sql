-- +migrate Up
CREATE TABLE IF NOT EXISTS "hatchery" (
    id uuid PRIMARY KEY,
    name TEXT NOT NULL,
    config JSONB,
    last_heartbeat TIMESTAMP WITH TIME ZONE DEFAULT LOCALTIMESTAMP,
    sig BYTEA,
    signer TEXT
);
SELECT create_unique_index('hatchery', 'idx_unq_hatchery', 'name');

CREATE TABLE IF NOT EXISTS "rbac_hatchery" (
    id  BIGSERIAL PRIMARY KEY,
    rbac_uuid uuid NOT NULL,
    role VARCHAR(255),
    hatchery_id BIGINT NOT NULL,
    region_id uuid NOT NULL,
    sig BYTEA,
    signer TEXT
);
SELECT create_foreign_key_idx_cascade('FK_rbac_hatchery', 'rbac_hatchery', 'rbac', 'rbac_uuid', 'id');
SELECT create_foreign_key('FK_rbac_hatchery_id', 'rbac_hatchery', 'hatchery', 'hatchery_id', 'id');
SELECT create_foreign_key('FK_rbac_hatchery_id', 'rbac_hatchery', 'region', 'region_id', 'id');
SELECT create_index('rbac_hatchery', 'idx_rbac_hatchery_role', 'role');
SELECT create_index('rbac_hatchery', 'idx_rbac_hatchery_region', 'region_id');
SELECT create_index('rbac_hatchery', 'idx_rbac_hatchery_hatchery', 'hatchery_id');

CREATE TABLE rbac_hatchery_users
(
    "id"             BIGSERIAL PRIMARY KEY,
    "rbac_hatchery_id" BIGINT,
    "user_id"        character varying(36),
    sig              BYTEA,
    signer           TEXT
);
SELECT create_foreign_key_idx_cascade('FK_rbac_hatchery_users', 'rbac_hatchery_users', 'rbac_hatchery', 'rbac_hatchery_id', 'id');
SELECT create_foreign_key_idx_cascade('FK_rbac_hatchery_users_ids', 'rbac_hatchery_users', 'authentified_user', 'user_id', 'id');
SELECT create_unique_index('rbac_hatchery_users', 'idx_unq_rbac_hatchery_users', 'rbac_hatchery_id,user_id');

CREATE TABLE rbac_hatchery_groups
(
    "id"             BIGSERIAL PRIMARY KEY,
    "rbac_hatchery_id" BIGINT,
    "group_id"       BIGINT,
    sig              BYTEA,
    signer           TEXT
);
SELECT create_foreign_key_idx_cascade('FK_rbac_hatchery_groups', 'rbac_hatchery_groups', 'rbac_hatchery', 'rbac_hatchery_id', 'id');
SELECT create_foreign_key_idx_cascade('FK_rbac_hatchery_groups_ids', 'rbac_hatchery_groups', 'group', 'group_id', 'id');
SELECT create_unique_index('rbac_hatchery_groups', 'idx_unq_rbac_hatchery_groups', 'rbac_hatchery_id,group_id');

-- +migrate Down
DROP TABLE rbac_hatchery;
DROP TABLE hatchery;

