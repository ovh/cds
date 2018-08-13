-- +migrate Up
ALTER TABLE workflow_node_run ADD COLUMN vcs_tag TEXT;

-- +migrate Down
ALTER TABLE workflow_node_run DROP COLUMN vcs_tag;
