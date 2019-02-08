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

ALTER TABLE environment_group DROP CONSTRAINT fk_environment_group_env;
ALTER TABLE environment_group DROP CONSTRAINT fk_environment_group_group;

ALTER TABLE pipeline_group DROP CONSTRAINT fk_pipeline_group;
ALTER TABLE pipeline_group DROP CONSTRAINT fk_pipeline_group_pipeline;

ALTER TABLE application_group DROP CONSTRAINT fk_application_group_group;
ALTER TABLE application_group DROP CONSTRAINT fk_application_group_application;

ALTER TABLE workflow_group DROP CONSTRAINT fk_workflow_group_group;
ALTER TABLE workflow_group DROP CONSTRAINT fk_workflow_group_workflow;

-- +migrate Down
DROP TABLE "workflow_node_group";
DROP TABLE "workflow_perm";

-- PIPELINE GROUP
select create_foreign_key('FK_PIPELINE_GROUP_PIPELINE', 'pipeline_group', 'pipeline', 'pipeline_id', 'id');
select create_foreign_key('FK_PIPELINE_GROUP', 'pipeline_group', 'group', 'group_id', 'id');

-- ENVIRONMENT_GROUP
select create_foreign_key('FK_ENVIRONMENT_GROUP_ENV', 'environment_group', 'environment', 'environment_id', 'id');
select create_foreign_key('FK_ENVIRONMENT_GROUP_GROUP', 'environment_group', 'group', 'group_id', 'id');

-- APPLICATION GROUP
select create_foreign_key('FK_APPLICATION_GROUP_APPLICATION', 'application_group', 'application', 'application_id', 'id');
select create_foreign_key('FK_APPLICATION_GROUP_GROUP', 'application_group', 'group', 'group_id', 'id');

-- PREVIOUS WORKFLOW GROUP
SELECT create_foreign_key_idx_cascade('FK_WORKFLOW_GROUP_WORKFLOW', 'workflow_group', 'workflow', 'workflow_id', 'id');
SELECT create_foreign_key_idx_cascade('FK_WORKFLOW_GROUP_GROUP', 'workflow_group', 'group', 'group_id', 'id');