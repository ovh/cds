-- +migrate Up
ALTER TABLE user_gpg_key ADD COLUMN sub_key_ids JSONB NOT NULL DEFAULT '[]'::jsonb;
CREATE INDEX IDX_USER_GPG_KEY_SUB_KEY_IDS ON user_gpg_key USING GIN (sub_key_ids);

-- +migrate Down
DROP INDEX IF EXISTS IDX_USER_GPG_KEY_SUB_KEY_IDS;
ALTER TABLE user_gpg_key DROP COLUMN sub_key_ids;
