-- +migrate Up
CREATE TABLE IF NOT EXISTS "workflow_perm" (workflow_id INT, project_group_id INT, role INT, PRIMARY KEY(project_group_id, workflow_id));
SELECT create_foreign_key_idx_cascade('FK_WORKFLOW_GROUP_WORKFLOW', 'workflow_perm', 'workflow', 'workflow_id', 'id');
SELECT create_foreign_key_idx_cascade('FK_WORKFLOW_GROUP_PROJECT_GROUP', 'workflow_perm', 'project_group', 'project_group_id', 'id');

CREATE TABLE IF NOT EXISTS "workflow_node_group" (id BIGSERIAL, workflow_node_id INT, workflow_group_id INT, role INT, PRIMARY KEY(project_group_id, workflow_node_id));
SELECT create_foreign_key_idx_cascade('FK_WORKFLOW_NODE_GROUP_W_NODE', 'workflow_node_group', 'w_node', 'workflow_node_id', 'id');
SELECT create_foreign_key_idx_cascade('FK_WORKFLOW_NODE_GROUP_WORKFLOW_GROUP', 'workflow_node_group', 'workflow_perm', 'workflow_group_id', 'id');
-- TODO ADD DELETE ON CASCADE
-- +migrate Down
DROP TABLE "workflow_node_group";
DROP TABLE "workflow_group";