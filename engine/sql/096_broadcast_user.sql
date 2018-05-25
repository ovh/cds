-- +migrate Up
CREATE TABLE broadcast_read (
  broadcast_id BIGINT,
  user_id BIGINT,
  PRIMARY KEY (broadcast_id, user_id)
);

SELECT create_foreign_key_idx_cascade('FK_BROADCAST_READ_BROADCAST', 'broadcast_read', 'broadcast', 'broadcast_id', 'id');
SELECT create_foreign_key_idx_cascade('FK_BROADCAST_READ_USER', 'broadcast_read', 'user', 'user_id', 'id');

-- +migrate Down
DROP TABLE broadcast_read;
