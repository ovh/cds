-- +migrate Up
ALTER TABLE workflow_node ADD CONSTRAINT "UNIQ_WORKFLOW_NODE_NAME" UNIQUE (workflow_id, name);

-- +migrate Down
ALTER TABLE workflow_node DROP CONSTRAINT "UNIQ_WORKFLOW_NODE_NAME";
