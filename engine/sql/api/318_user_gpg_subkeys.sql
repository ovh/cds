-- +migrate Up
ALTER TABLE user_gpg_key ADD COLUMN sub_keys JSONB;
CREATE INDEX idx_gin_user_gpg_subkeys ON user_gpg_key USING GIN (sub_keys);

-- +migrate Down
DROP INDEX IF EXISTS idx_gin_user_gpg_subkeys;
ALTER TABLE user_gpg_key DROP COLUMN sub_keys;
