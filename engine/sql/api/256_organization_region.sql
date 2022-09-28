-- +migrate Up
CREATE TABLE IF NOT EXISTS "organization_region" (
    id uuid PRIMARY KEY,
    organization_id uuid NOT NULL,
    region_id uuid NOT NULL,
    sig BYTEA,
    signer TEXT
);
SELECT create_foreign_key_idx_cascade('FK_ORGANIZATION_REGION_REGION', 'organization_region', 'region', 'region_id', 'id');
SELECT create_foreign_key_idx_cascade('FK_ORGANIZATION_REGION_ORGANIZATION', 'organization_region', 'organization', 'organization_id', 'id');
SELECT create_unique_index('organization_region', 'idx_unq_organization_region', 'organization_id,region_id');

-- +migrate Down
DROP TABLE organization_region;
