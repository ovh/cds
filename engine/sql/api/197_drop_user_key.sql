-- +migrate Up

DROP TABLE user_key;

-- +migrate Down

SELECT 1;
