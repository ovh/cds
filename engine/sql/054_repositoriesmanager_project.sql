-- +migrate Up
ALTER TABLE project ADD COLUMN vcs_servers BYTEA;

-- +migrate Down
DROP TABLE project DROP COLUMN vcs_servers;