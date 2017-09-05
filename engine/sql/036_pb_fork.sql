-- +migrate Up
ALTER TABLE pipeline_build ADD COLUMN vcs_remote_url TEXT;
ALTER TABLE pipeline_build ADD COLUMN vcs_remote TEXT;

-- +migrate Down
ALTER TABLE pipeline_build DROP COLUMN vcs_remote_url;
ALTER TABLE pipeline_build DROP COLUMN vcs_remote;
