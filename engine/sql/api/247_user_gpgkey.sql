-- +migrate Up
CREATE TABLE IF NOT EXISTS "user_gpg_key" (
    "id" uuid PRIMARY KEY,
    "authentified_user_id" VARCHAR(36) NOT NULL,
    "key_id" VARCHAR(255) NOT NULL,
    "public_key" TEXT NOT NULL,
    "sig"           BYTEA,
    "signer"        TEXT
);
SELECT create_unique_index('user_gpg_key', 'idx_unq_key_id', 'key_id');
SELECT create_foreign_key_idx_cascade('fk_gpg_key_user', 'user_gpg_key', 'authentified_user', 'authentified_user_id', 'id');

-- +migrate Down
DROP TABLE user_gpg_key;
