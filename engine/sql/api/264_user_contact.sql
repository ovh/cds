-- +migrate Up
CREATE TABLE user_link (
  id BIGSERIAL PRIMARY KEY,
  authentified_user_id VARCHAR(36) NOT NULL,
  type VARCHAR(255) NOT NULL,
  username TEXT NOT NULL,
  created TIMESTAMP WITH TIME ZONE DEFAULT LOCALTIMESTAMP,
  sig BYTEA,
  signer TEXT
);
SELECT create_foreign_key_idx_cascade('FK_user_link', 'user_link', 'authentified_user', 'authentified_user_id', 'id');
SELECT create_unique_index('user_link', 'idx_unq_user_link', 'authentified_user_id,type');
SELECT create_unique_index('user_link', 'idx_unq_user_link_username', 'type,username');

-- +migrate Down
DROP TABLE user_link;
