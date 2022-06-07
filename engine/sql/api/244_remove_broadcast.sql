-- +migrate Up
DROP TABLE broadcast_read;
DROP TABLE broadcast;

-- +migrate Down
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

CREATE TABLE broadcast_read (
                                broadcast_id BIGINT,
                                authentified_user_id VARCHAR(36),
                                PRIMARY KEY (broadcast_id, user_id)
);
SELECT create_foreign_key_idx_cascade('FK_BROADCAST_READ_BROADCAST', 'broadcast_read', 'broadcast', 'broadcast_id', 'id');
SELECT create_foreign_key_idx_cascade('FK_BROADCAST_READ_AUTHENTIFIED_USER', 'broadcast_read', 'authentified_user', 'authentified_user_id', 'id');
