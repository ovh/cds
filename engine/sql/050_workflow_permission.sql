-- +migrate Up
SELECT create_foreign_key_idx_cascade('FK_WORKFLOW_GROUP_WORKFLOW', 'workflow_group', 'workflow', 'workflow_id', 'id');
SELECT create_foreign_key_idx_cascade('FK_WORKFLOW_GROUP_GROUP', 'workflow_group', 'group', 'group_id', 'id');

ALTER TABLE workflow ADD COLUMN metadata JSONB;

-- +migrate Down
ALTER TABLE workflow_group DROP CONSTRAINT FK_WORKFLOW_GROUP_WORKFLOW;
ALTER TABLE workflow_group DROP CONSTRAINT FK_WORKFLOW_GROUP_GROUP;

ALTER table workflow DROP COLUMN metadata;