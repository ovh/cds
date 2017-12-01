-- +migrate Up

CREATE TABLE IF NOT EXISTS "workflow_node_join" (
    id BIGSERIAL PRIMARY KEY,
    workflow_id BIGINT NOT NULL
);

CREATE TABLE IF NOT EXISTS "workflow_node_join_source" (
    workflow_node_join_id BIGSERIAL,
    workflow_node_id BIGINT NOT NULL,
    PRIMARY KEY(workflow_node_join_id, workflow_node_id)
);

SELECT create_foreign_key('FK_WORKFLOW_NODE_JOIN_WORKFLOW', 'workflow_node_join', 'workflow', 'workflow_id', 'id');
SELECT create_foreign_key('FK_WORKFLOW_NODE_JOIN_SOURCE', 'workflow_node_join_source', 'workflow_node_join', 'workflow_node_join_id', 'id');
SELECT create_foreign_key('FK_WORKFLOW_NODE_JOIN_SOURCE_NODE', 'workflow_node_join_source', 'workflow_node', 'workflow_node_id', 'id');


CREATE TABLE IF NOT EXISTS "workflow_node_join_trigger" (
    id BIGSERIAL PRIMARY KEY,
    workflow_node_join_id BIGINT NOT NULL,
    workflow_dest_node_id BIGINT NOT NULL,
    conditions JSONB
);

SELECT create_foreign_key('FK_WORKFLOW_TRIGGER_WORKFLOW_NODE', 'workflow_node_trigger', 'workflow_node_join', 'workflow_node_join_id', 'id');
SELECT create_foreign_key('FK_WORKFLOW_TRIGGER_WORKFLOW_NODE', 'workflow_node_trigger', 'workflow_node', 'workflow_dest_node_id', 'id');

-- +migrate Down

DROP TABLE workflow_node_join_trigger CASCADE;
DROP TABLE workflow_node_join_source CASCADE;
DROP TABLE workflow_node_join CASCADE;
