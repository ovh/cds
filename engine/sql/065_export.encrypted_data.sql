-- +migrate Up
CREATE TABLE IF NOT EXISTS "encrypted_data" (
    project_id BIGINT NOT NULL,
    token VARCHAR(32) NOT NULL,
    content_name VARCHAR(32) NOT NULL, 
    encrypted_content BYTEA NOT NULL,
    PRIMARY KEY(project_id, content_name)
);

SELECT create_foreign_key_idx_cascade('FK_ENCRYPTED_PROJECT', 'encrypted_data', 'project', 'project_id', 'id');

-- +migrate Down
DROP TABLE encrypted_data;
