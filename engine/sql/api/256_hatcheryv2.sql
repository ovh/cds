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

-- +migrate Down
DROP TABLE hatchery;
