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
    region_id uuid,
    sig BYTEA,
    signer TEXT
);
SELECT create_foreign_key_idx_cascade('FK_rbac_hatchery', 'rbac_hatchery', 'rbac', 'rbac_uuid', 'id');
SELECT create_index('rbac_hatchery', 'idx_rbac_hatchery_role', 'role');

CREATE TABLE IF NOT EXISTS "rbac_hatchery_workflow" (
    id  BIGSERIAL PRIMARY KEY,
    rbac_hatchery_id BIGINT NOT NULL,
    pattern TEXT,
    sig BYTEA,
    signer TEXT
);
SELECT create_foreign_key_idx_cascade('FK_rbac_hatchery_workflow', 'rbac_hatchery_workflow', 'rbac_hatchery_workflow', 'rbac_hatchery_id', 'id');

ALTER TABLE region ADD COLUMN by_default BOOLEAN DEFAULT false;

-- +migrate Down
DROP TABLE rbac_hatchery_workflow;
DROP TABLE rbac_hatchery;
DROP TABLE hatchery;

ALTER TABLE region DROP COLUMN by_default;
