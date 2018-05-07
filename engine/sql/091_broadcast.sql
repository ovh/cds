-- +migrate Up
CREATE TABLE broadcast (
  id BIGSERIAL PRIMARY KEY,
  title VARCHAR(256) NOT NULL default '',
  content TEXT NOT NULL default '',
  level VARCHAR(10) NOT NULL default '',
  created TIMESTAMP WITH TIME ZONE,
  updated TIMESTAMP WITH TIME ZONE,
  archived BOOLEAN default false,
  project_id BIGINT NULL
);

SELECT create_foreign_key_idx_cascade('FK_BROADCAST_PROJECT', 'broadcast', 'project', 'project_id', 'id');

-- +migrate Down
DROP TABLE broadcast;
