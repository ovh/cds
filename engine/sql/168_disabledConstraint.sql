-- +migrate Up
ALTER TABLE workflow DROP CONSTRAINT IF EXISTS fk_workflow_root_node;
ALTER TABLE workflow_node DROP CONSTRAINT IF EXISTS fk_workflow_node_pipeline;
ALTER TABLE workflow_node DROP CONSTRAINT IF EXISTS  fk_workflow_node_workflow;

ALTER TABLE workflow_notification_source DROP CONSTRAINT IF EXISTS fk_workflow_notification_source_node;
ALTER TABLE workflow_notification_source DROP CONSTRAINT workflow_notification_source_pkey;
ALTER TABLE workflow_notification_source ADD PRIMARY KEY (workflow_notification_id, node_id);
ALTER TABLE workflow_notification_source ALTER COLUMN workflow_node_id DROP NOT NULL;

-- +migrate Down
SELECT 1;

