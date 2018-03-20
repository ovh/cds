-- +migrate Up
ALTER TABLE workflow_node_run ADD COLUMN vcs_repository text;
ALTER TABLE workflow_node_run ADD COLUMN vcs_hash text;
ALTER TABLE workflow_node_run ADD COLUMN vcs_branch text;
-- +migrate Down
ALTER TABLE workflow_node_run DROP COLUMN vcs_repository;
ALTER TABLE workflow_node_run DROP COLUMN vcs_hash;
ALTER TABLE workflow_node_run DROP COLUMN vcs_branch;
