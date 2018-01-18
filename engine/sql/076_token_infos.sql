-- +migrate Up
ALTER TABLE token ADD COLUMN description TEXT;
ALTER TABLE token ADD COLUMN creator TEXT;
ALTER TABLE token ADD COLUMN id BIGSERIAL PRIMARY KEY;
SELECT create_foreign_key_idx_cascade('FK_TOKEN_GROUP', 'token', 'group', 'group_id', 'id');
-- +migrate Down
ALTER TABLE token DROP CONSTRAINT FK_TOKEN_GROUP;
ALTER TABLE token DROP COLUMN description;
ALTER TABLE token DROP COLUMN creator;
ALTER TABLE token DROP COLUMN id;
