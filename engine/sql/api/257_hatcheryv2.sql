-- +migrate Up
CREATE TABLE IF NOT EXISTS "hatchery" (
    id uuid PRIMARY KEY,
    name TEXT NOT NULL,
    config JSONB,
    sig BYTEA,
    signer TEXT
);
SELECT create_unique_index('hatchery', 'idx_unq_hatchery', 'name');

CREATE TABLE IF NOT EXISTS "rbac_hatchery" (
    id  BIGSERIAL PRIMARY KEY,
    rbac_id uuid NOT NULL,
    role VARCHAR(255),
    hatchery_id uuid NOT NULL,
    region_id uuid NOT NULL,
    sig BYTEA,
    signer TEXT
);
SELECT create_foreign_key_idx_cascade('FK_rbac_hatchery', 'rbac_hatchery', 'rbac', 'rbac_id', 'id');
SELECT create_foreign_key('FK_rbac_hatchery_id', 'rbac_hatchery', 'hatchery', 'hatchery_id', 'id');
SELECT create_foreign_key('FK_rbac_hatchery_id', 'rbac_hatchery', 'region', 'region_id', 'id');
SELECT create_unique_index('rbac_hatchery', 'idx_unq_rbac_hatchery', 'hatchery_id');
SELECT create_index('rbac_hatchery', 'idx_rbac_hatchery_role', 'role');
SELECT create_index('rbac_hatchery', 'idx_rbac_hatchery_region', 'region_id');

-- +migrate Down
DROP TABLE rbac_hatchery;
DROP TABLE hatchery;

