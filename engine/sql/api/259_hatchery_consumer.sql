-- +migrate Up
CREATE TABLE IF NOT EXISTS "auth_consumer_hatchery" (
    id  uuid PRIMARY KEY,
    auth_consumer_id VARCHAR(36) NOT NULL,
    hatchery_id uuid NOT NULL,
    sig BYTEA,
    signer TEXT
);
SELECT create_foreign_key_idx_cascade('FK_auth_consumer_hatchery', 'auth_consumer_hatchery', 'auth_consumer', 'auth_consumer_id', 'id');
SELECT create_foreign_key_idx_cascade('FK_auth_consumer_hatchery_id', 'auth_consumer_hatchery', 'hatchery', 'hatchery_id', 'id');

ALTER TABLE "hatchery" ADD COLUMN "last_heartbeat" TIMESTAMP WITH TIME ZONE;
ALTER TABLE "hatchery" ADD COLUMN "public_key" BYTEA;
ALTER TABLE "hatchery" ADD COLUMN "http_url" TEXT;

CREATE TABLE IF NOT EXISTS "hatchery_status" (
    id BIGSERIAL PRIMARY KEY,
    hatchery_id uuid,
    session_id VARCHAR(36),
    monitoring_status JSONB
);
SELECT create_foreign_key_idx_cascade('FK_hatchery_status_hatchery', 'hatchery_status', 'hatchery', 'hatchery_id', 'id');
SELECT create_foreign_key_idx_cascade('FK_hatchery_status_session', 'hatchery_status', 'auth_session', 'session_id', 'id');
SELECT create_unique_index('hatchery_status', 'idx_hatchery_status_uniq', 'session_id,hatchery_id');


-- +migrate Down
DROP TABLE auth_consumer_hatchery;
DROP TABLE hatchery_status;
ALTER TABLE "hatchery" DROP COLUMN "last_heartbeat";
ALTER TABLE "hatchery" DROP COLUMN "public_key";
ALTER TABLE "hatchery" DROP COLUMN "http_url";


