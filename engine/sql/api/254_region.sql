-- +migrate Up
CREATE TABLE IF NOT EXISTS "region" (
    "id" uuid PRIMARY KEY,
    "name" TEXT,
    "sig"           BYTEA,
    "signer"        TEXT
);
SELECT create_unique_index('region', 'idx_unq_region_name', 'name');

-- +migrate Down
DROP TABLE region;
