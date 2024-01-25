-- +migrate Up
ALTER TABLE v2_workflow_run ADD COLUMN vcs_server VARCHAR(255) NOT NULL DEFAULT '';
ALTER TABLE v2_workflow_run ADD COLUMN repository VARCHAR(255) NOT NULL DEFAULT '';

-- +migrate Down
ALTER TABLE v2_workflow_run DROP COLUMN vcs_server;
ALTER TABLE v2_workflow_run DROP COLUMN repository;