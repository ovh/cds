-- +migrate Up
CREATE TABLE access_token
(
    id BIGSERIAL PRIMARY KEY,
    description VARCHAR(255) NOT NULL,
    user_id BIGINT,
    created TIMESTAMP WITH TIME ZONE DEFAULT LOCALTIMESTAMP,
    expired_at TIMESTAMP WITH TIME ZONE,
    status VARCHAR(25),
    origin  VARCHAR(25)
);

CREATE TABLE access_token_group
(
    access_token_id BIGINT,
    group_id BIGINT
)


SELECT create_foreign_key_idx_cascade('FK_ACCESS_TOKEN_USER', 'access_token', 'user', 'user_id', 'id');
SELECT create_foreign_key_idx_cascade('FK_ACCESS_TOKEN_USER', 'access_token', 'group', 'group_id', 'id');

-- +migrate Down
DROP TABLE access_token_group;
DROP TABLE access_token;