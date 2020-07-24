-- +migrate Up
ALTER TABLE workflow_notification_source ADD COLUMN node_id BIGINT;
SELECT create_foreign_key_idx_cascade('FK_WORKFLOW_NOTIFICATION_NODE', 'workflow_notification_source', 'w_node', 'node_id', 'id');

-- +migrate Down

ALTER TABLE workflow_notification_source DROP COLUMN node_id;
