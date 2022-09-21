-- +migrate Up
CREATE TABLE IF NOT EXISTS "organization" (
    "id" uuid PRIMARY KEY,
    "name" TEXT,
    "sig"           BYTEA,
    "signer"        TEXT
);
SELECT create_unique_index('organization', 'idx_unq_organization_name', 'name');


-- +migrate Down
DROP TABLE organization;
