-- +migrate Up
select create_unique_index('workflow_node_run', 'IDX_WORKFLOW_NODE_RUN_SUBNUM', 'workflow_node_id,num,sub_num');

-- +migrate Down
DROP INDEX IDX_WORKFLOW_NODE_RUN_SUBNUM;
