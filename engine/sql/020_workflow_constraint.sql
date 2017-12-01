-- +migrate Up

-- workflow_node :Add trigger src id
ALTER TABLE workflow_node ADD COLUMN workflow_trigger_src_id BIGINT;
ALTER TABLE workflow_node ADD COLUMN workflow_trigger_join_src_id BIGINT;

-- workflow_node : Add constaint with cascade
ALTER TABLE workflow_node
ADD CONSTRAINT FK_WORKFLOW_NODE_TRIGGER
FOREIGN KEY(workflow_trigger_src_id) REFERENCES workflow_node_trigger(id) ON DELETE CASCADE DEFERRABLE INITIALLY DEFERRED;
ALTER TABLE workflow_node
ADD CONSTRAINT FK_WORKFLOW_NODE_TRIGGER_JOIN
FOREIGN KEY(workflow_trigger_join_src_id) REFERENCES workflow_node_join_trigger(id) ON DELETE CASCADE DEFERRABLE INITIALLY DEFERRED;

-- workflow_node_trigger: add constraint
SELECT create_foreign_key_idx_cascade('FK_WORKFLOW_TRIGGER_WORKFLOW_NODE_DEST', 'workflow_node_trigger', 'workflow_node', 'workflow_dest_node_id', 'id');

-- workflow_node_join
ALTER TABLE workflow_node_join DROP CONSTRAINT fk_workflow_node_join_workflow;
SELECT create_foreign_key_idx_cascade('FK_WORKFLOW_NODE_JOIN_WORKFLOW', 'workflow_node_join', 'workflow', 'workflow_id', 'id');

-- workflow_node_join_source
ALTER TABLE workflow_node_join_source DROP CONSTRAINT FK_WORKFLOW_NODE_JOIN_SOURCE;
SELECT create_foreign_key_idx_cascade('FK_WORKFLOW_NODE_JOIN_SOURCE', 'workflow_node_join_source', 'workflow_node_join', 'workflow_node_join_id', 'id');

-- workflow_node_join_trigger
SELECT create_foreign_key_idx_cascade('FK_WORKFLOW_TRIGGER_JOIN_WORKFLOW_NODE', 'workflow_node_join_trigger', 'workflow_node_join', 'workflow_node_join_id', 'id');
SELECT create_foreign_key_idx_cascade('FK_WORKFLOW_TRIGGER_JOIN_WORKFLOW_NODE_DEST', 'workflow_node_join_trigger', 'workflow_node', 'workflow_dest_node_id', 'id');

-- +migrate Down
ALTER TABLE workflow_node DROP COLUMN workflow_trigger_src_id;
ALTER TABLE workflow_node DROP COLUMN workflow_trigger_join_src_id;

ALTER TABLE workflow_node_trigger DROP CONSTRAINT FK_WORKFLOW_TRIGGER_WORKFLOW_NODE_DEST;

ALTER TABLE workflow_node_join DROP CONSTRAINT fk_workflow_node_join_workflow;
SELECT create_foreign_key('FK_WORKFLOW_NODE_JOIN_WORKFLOW', 'workflow_node_join', 'workflow', 'workflow_id', 'id');

ALTER TABLE workflow_node_join_source DROP CONSTRAINT FK_WORKFLOW_NODE_JOIN_SOURCE;
SELECT create_foreign_key('FK_WORKFLOW_NODE_JOIN_SOURCE', 'workflow_node_join_source', 'workflow_node_join', 'workflow_node_join_id', 'id');

ALTER TABLE workflow_node_join_trigger DROP CONSTRAINT FK_WORKFLOW_TRIGGER_JOIN_WORKFLOW_NODE;
ALTER TABLE workflow_node_join_trigger DROP CONSTRAINT FK_WORKFLOW_TRIGGER_JOIN_WORKFLOW_NODE_DEST;