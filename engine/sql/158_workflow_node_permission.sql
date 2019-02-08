-- +migrate Up
SELECT create_unique_index('project_group','IDX_PROJECT_GROUP_ID','id');
CREATE TABLE IF NOT EXISTS "workflow_perm" (id BIGSERIAL, workflow_id BIGINT, project_group_id BIGINT, role INT, PRIMARY KEY(project_group_id, workflow_id));
SELECT create_foreign_key_idx_cascade('FK_WORKFLOW_PERM_WORKFLOW', 'workflow_perm', 'workflow', 'workflow_id', 'id');
SELECT create_foreign_key_idx_cascade('FK_WORKFLOW_PERM_PROJECT_GROUP', 'workflow_perm', 'project_group', 'project_group_id', 'id');
SELECT create_unique_index('workflow_perm','IDX_WORKFLOW_PERM_ID','id');

CREATE TABLE IF NOT EXISTS "workflow_node_group" (id BIGSERIAL, workflow_node_id BIGINT, workflow_group_id BIGINT, role INT, PRIMARY KEY(workflow_node_id, workflow_group_id));
SELECT create_foreign_key_idx_cascade('FK_WORKFLOW_NODE_GROUP_W_NODE', 'workflow_node_group', 'w_node', 'workflow_node_id', 'id');
SELECT create_foreign_key_idx_cascade('FK_WORKFLOW_NODE_GROUP_WORKFLOW_GROUP', 'workflow_node_group', 'workflow_perm', 'workflow_group_id', 'id');
SELECT create_unique_index('workflow_node_group','IDX_WORKFLOW_NODE_GROUP_ID','id');

-- TODO drop constraint on old table _group
-- +migrate Down
DROP TABLE "workflow_node_group";
DROP TABLE "workflow_perm";