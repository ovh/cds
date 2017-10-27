-- +migrate Up
ALTER TABLE project ADD COLUMN vcs_servers BYTEA;
ALTER TABLE application ADD COLUMN vcs_server VARCHAR(256);

-- +migrate Down
DROP TABLE project DROP COLUMN vcs_servers;
ALTER TABLE application DROP COLUMN vcs_server;
