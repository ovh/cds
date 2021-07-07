-- +migrate Up
CREATE INDEX idx_workflow_node_run_workflow_id ON workflow_node_run (workflow_id, vcs_hash, workflow_node_name, num);

-- +migrate Down
DROP INDEX idx_workflow_node_run_workflow_id;
