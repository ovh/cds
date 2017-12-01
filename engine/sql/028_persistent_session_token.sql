-- +migrate Up
CREATE TABLE IF NOT EXISTS "user_persistent_session" (
  token VARCHAR(36) PRIMARY KEY, 
  creation_date TIMESTAMP WITH TIME ZONE DEFAULT LOCALTIMESTAMP,
  last_connection_date TIMESTAMP WITH TIME ZONE DEFAULT LOCALTIMESTAMP,
  comment VARCHAR(256) NOT NULL,
  user_id BIGINT NOT NULL
);

SELECT create_foreign_key_idx_cascade('FK_USER_PERSISTENT_SESSION_USER', 'user_persistent_session', 'user', 'user_id', 'id');


-- +migrate Down
DROP TABLE user_persistent_session;
