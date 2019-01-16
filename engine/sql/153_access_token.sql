-- +migrate Up
CREATE TABLE access_token
(
    id VARCHAR(64) PRIMARY KEY,
    description VARCHAR(255) NOT NULL,
    user_id BIGINT,
    created TIMESTAMP WITH TIME ZONE DEFAULT LOCALTIMESTAMP,
    expired_at TIMESTAMP WITH TIME ZONE,
    status VARCHAR(25),
    origin  VARCHAR(25)
);

CREATE TABLE access_token_group
(
    access_token_id VARCHAR(255),
    group_id BIGINT,
    PRIMARY KEY (access_token_id, group_id)
);


SELECT create_foreign_key_idx_cascade('FK_ACCESS_TOKEN_USER', 'access_token', 'user', 'user_id', 'id');
SELECT create_foreign_key_idx_cascade('FK_ACCESS_TOKEN_GROUP_ACCESS_TOKEN', 'access_token_group', 'access_token', 'access_token_id', 'id');
SELECT create_foreign_key_idx_cascade('FK_ACCESS_TOKEN_GROUP_GROUP', 'access_token_group', 'group', 'group_id', 'id');
SELECT create_unique_index('access_token', 'IDX_ACCESS_TOKEN', 'user_id,description');

-- +migrate Down
DROP TABLE access_token_group;
DROP TABLE access_token;