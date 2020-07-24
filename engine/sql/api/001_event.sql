-- +migrate Up

DROP TABLE  "user_notification";

-- +migrate Down

CREATE TABLE IF NOT EXISTS "user_notification" (id BIGSERIAL PRIMARY KEY, type TEXT, content JSONB, status TEXT, creation_date INT);
