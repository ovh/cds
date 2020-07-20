-- +migrate Up
DELETE FROM token WHERE group_id NOT IN (SELECT id FROM "group");
SELECT create_foreign_key_idx_cascade('FK_TOKEN_GROUP', 'token', 'group', 'group_id', 'id');
-- +migrate Down
ALTER TABLE token DROP CONSTRAINT FK_TOKEN_GROUP;
