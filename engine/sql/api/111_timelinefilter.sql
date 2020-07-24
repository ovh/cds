-- +migrate Up
CREATE TABLE IF NOT EXISTS "user_timeline" (
    user_id BIGINT,
    filter JSONB
);

SELECT create_foreign_key_idx_cascade('FK_USER_TIMELINE', 'user_timeline', 'user', 'user_id', 'id');

-- +migrate Down
DROP TABLE user_timeline;
