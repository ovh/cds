-- +migrate Up
CREATE TABLE IF NOT EXISTS "workflow_notification" (
    id BIGSERIAL PRIMARY KEY,
    workflow_id BIGINT NOT NULL,
    type VARCHAR(50),
    settings JSONB
);

CREATE TABLE IF NOT EXISTS "workflow_notification_source" (
    workflow_notification_id BIGINT,
    workflow_node_id BIGINT NOT NULL,
    PRIMARY KEY(workflow_notification_id, workflow_node_id)
);

SELECT create_foreign_key_idx_cascade('FK_WORKFLOW_NOTIFICATION', 'workflow_notification', 'workflow', 'workflow_id', 'id');
SELECT create_foreign_key_idx_cascade('FK_WORKFLOW_NOTIFICATION_SOURCE_NODE', 'workflow_notification_source', 'workflow_node', 'workflow_node_id', 'id');
SELECT create_foreign_key_idx_cascade('FK_WORKFLOW_NOTIFICATION_SOURCE', 'workflow_notification_source', 'workflow_notification', 'workflow_notification_id', 'id');

-- +migrate Down
DROP TABLE workflow_notification CASCADE;
DROP TABLE workflow_notification_source CASCADE;
