-- +migrate Up
CREATE TABLE workflow_node_fork
(
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(255),
    workflow_node_id BIGINT NOT NULL
);
SELECT create_foreign_key_idx_cascade('FK_WORKFLOW_NODE_FORK', 'workflow_node_fork', 'workflow_node', 'workflow_node_id', 'id');

CREATE TABLE workflow_node_fork_trigger
(
    id BIGSERIAL PRIMARY KEY,
    workflow_node_fork_id BIGINT NOT NULL,
    workflow_dest_node_id BIGINT NOT NULL
);

SELECT create_unique_index('workflow_node_fork_trigger', 'IDX_NODE_FORK_TRIGGER_UNIQ', 'workflow_node_fork_id,workflow_dest_node_id');

SELECT create_foreign_key_idx_cascade('FK_WORKFLOW_NODE_FORK_TRIGGER_FORK', 'workflow_node_fork_trigger', 'workflow_node_fork', 'workflow_node_fork_id', 'id');
SELECT create_foreign_key_idx_cascade('FK_WORKFLOW_NODE_FORK_TRIGGER_NODE', 'workflow_node_fork_trigger', 'workflow_node', 'workflow_dest_node_id', 'id');


ALTER TABLE workflow_node ADD COLUMN workflow_fork_trigger_src_id BIGINT;
ALTER TABLE workflow_node ADD CONSTRAINT FK_WORKFLOW_NODE_TRIGGER_FORK FOREIGN KEY(workflow_fork_trigger_src_id) REFERENCES workflow_node_fork_trigger(id) ON DELETE CASCADE DEFERRABLE INITIALLY DEFERRED;


-- +migrate Down

DROP TABLE workflow_node_fork CASCADE;
DROP TABLE workflow_node_fork_trigger CASCADE;
ALTER TABLE workflow_node DROP COLUMN workflow_fork_trigger_src_id;
