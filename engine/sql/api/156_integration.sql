-- +migrate Up

CREATE TABLE integration_model AS TABLE platform_model;

CREATE SEQUENCE integration_model_id_seq;
ALTER TABLE integration_model ALTER COLUMN id SET NOT NULL;
ALTER TABLE integration_model ALTER COLUMN id SET DEFAULT nextval('integration_model_id_seq');
ALTER SEQUENCE integration_model_id_seq OWNED BY integration_model.id;
SELECT setval('"integration_model_id_seq"'::regclass, (SELECT MAX("id") FROM "integration_model"));

SELECT create_unique_index('integration_model', 'IDX_INTEGRATION_MODEL_NAME', 'name');
SELECT create_primary_key('integration_model', 'id');

CREATE TABLE project_integration AS TABLE project_platform;

CREATE SEQUENCE project_integration_id_seq;
ALTER TABLE project_integration ALTER COLUMN id SET NOT NULL;
ALTER TABLE project_integration ALTER COLUMN id SET DEFAULT nextval('project_integration_id_seq');
ALTER SEQUENCE project_integration_id_seq OWNED BY project_integration.id;
SELECT setval('"project_integration_id_seq"'::regclass, (SELECT MAX("id") FROM "project_integration"));

ALTER TABLE project_integration RENAME COLUMN platform_model_id TO integration_model_id;
SELECT create_primary_key('project_integration', 'id');

SELECT create_foreign_key_idx_cascade('fk_project_integration', 'project_integration', 'project', 'project_id', 'id');
SELECT create_unique_index('project_integration', 'IDX_PROJECT_INTEGRATION_NAME', 'project_id,name');
SELECT create_index('project_integration', 'IDX_PROJECT_INTEGRATION', 'project_id,name');

ALTER TABLE grpc_plugin ADD COLUMN integration_model_id BIGINT;
UPDATE grpc_plugin set integration_model_id = platform_model_id;

ALTER TABLE application_deployment_strategy ADD COLUMN project_integration_id BIGINT;
UPDATE application_deployment_strategy set project_integration_id = project_platform_id;
ALTER TABLE application_deployment_strategy DROP CONSTRAINT "application_deployment_strategy_pkey";
SELECT create_primary_key('application_deployment_strategy', 'application_id,project_integration_id');
SELECT create_foreign_key_idx_cascade('fk_application_deployment_strategy_integration', 'application_deployment_strategy', 'project_integration', 'project_integration_id', 'id');
ALTER TABLE application_deployment_strategy ALTER COLUMN project_platform_id DROP NOT NULL;
ALTER TABLE application_deployment_strategy DROP CONSTRAINT "fk_application_deployment_strategy_platform";

ALTER TABLE workflow_node_context ADD COLUMN project_integration_id BIGINT;
UPDATE workflow_node_context set project_integration_id = project_platform_id;
SELECT create_foreign_key('FK_WORKFLOW_NODE_PROJECT_INTEGRATION', 'workflow_node_context', 'project_integration', 'project_integration_id', 'id');
ALTER TABLE workflow_node_context ALTER COLUMN project_platform_id DROP NOT NULL;
ALTER TABLE workflow_node_context DROP CONSTRAINT "fk_workflow_node_project_platform";

ALTER TABLE w_node_context ADD COLUMN project_integration_id BIGINT;
UPDATE w_node_context set project_integration_id = project_platform_id;
SELECT create_foreign_key_idx_cascade('FK_W_NODE_CONTEXT_INTEGRATION', 'w_node_context', 'project_integration', 'project_integration_id', 'id');
ALTER TABLE w_node_context ALTER COLUMN project_platform_id DROP NOT NULL;
ALTER TABLE w_node_context DROP CONSTRAINT "fk_w_node_context_platform";

ALTER TABLE workflow_node_run_job ADD COLUMN integration_plugin_binaries JSONB;
UPDATE workflow_node_run_job set integration_plugin_binaries = platform_plugin_binaries;

UPDATE grpc_plugin set type = 'integration-deploy_application' WHERE type = 'platform' OR type = 'platform-deploy_application';

UPDATE workflow set workflow_data=replace(workflow_data::TEXT, '"platform_model_id":', '"integration_model_id":')::jsonb;
UPDATE workflow set workflow_data=replace(workflow_data::TEXT, '"project_platform_id":', '"project_integration_id":')::jsonb;

UPDATE workflow_run set workflow=replace(workflow::TEXT, '"platform_model_id":', '"integration_model_id":')::jsonb
WHERE workflow_id in (
SELECT workflow.id from workflow
JOIN w_node on workflow.id = w_node.workflow_id
JOIN w_node_context on w_node.id = w_node_context.node_id
AND w_node_context.project_platform_id > 0
GROUP BY workflow.id)
AND last_execution>NOW()- INTERVAL '5 DAY';

-- update project_platform and project_platforms attributes
UPDATE workflow_run set workflow=replace(workflow::TEXT, '"project_platform', '"project_integration')::jsonb
WHERE workflow_id in (
SELECT workflow.id from workflow
JOIN w_node on workflow.id = w_node.workflow_id
JOIN w_node_context on w_node.id = w_node_context.node_id
AND w_node_context.project_platform_id > 0
GROUP BY workflow.id)
AND last_execution>NOW()- INTERVAL '5 DAY';


-- +migrate Down

ALTER TABLE application_deployment_strategy DROP CONSTRAINT "application_deployment_strategy_pkey";
ALTER TABLE grpc_plugin DROP COLUMN integration_model_id;
ALTER TABLE application_deployment_strategy DROP COLUMN project_integration_id;
ALTER TABLE workflow_node_context DROP COLUMN project_integration_id;
ALTER TABLE w_node_context DROP COLUMN project_integration_id;
ALTER TABLE workflow_node_run_job DROP COLUMN integration_plugin_binaries;
SELECT create_foreign_key('FK_WORKFLOW_NODE_PROJECT_PLATFORM', 'workflow_node_context', 'project_platform', 'project_platform_id', 'id');
SELECT create_foreign_key_idx_cascade('FK_W_NODE_CONTEXT_PLATFORM', 'w_node_context', 'project_platform', 'project_platform_id', 'id');
SELECT create_primary_key('application_deployment_strategy', 'application_id,project_platform_id');
SELECT create_foreign_key_idx_cascade('fk_application_deployment_strategy_platform', 'application_deployment_strategy', 'project_platform', 'project_platform_id', 'id');
DROP TABLE integration_model;
DROP TABLE project_integration;

UPDATE grpc_plugin set type = 'platform-deploy_application' WHERE type = 'integration-deploy_application';

UPDATE workflow set workflow_data=replace(workflow_data::TEXT, '"integration_model_id":', '"platform_model_id":')::jsonb;
UPDATE workflow set workflow_data=replace(workflow_data::TEXT, '"project_integration_id":', '"project_platform_id":')::jsonb;

UPDATE workflow_run set workflow=replace(workflow::TEXT, '"integration_model_id":', '"platform_model_id":')::jsonb
WHERE workflow_id in (
SELECT workflow.id from workflow
JOIN w_node on workflow.id = w_node.workflow_id
JOIN w_node_context on w_node.id = w_node_context.node_id
AND w_node_context.project_platform_id > 0
GROUP BY workflow.id)
AND last_execution>NOW()- INTERVAL '5 DAY';

UPDATE workflow_run set workflow=replace(workflow::TEXT, '"project_integration', '"project_platform')::jsonb
WHERE workflow_id in (
SELECT workflow.id from workflow
JOIN w_node on workflow.id = w_node.workflow_id
JOIN w_node_context on w_node.id = w_node_context.node_id
AND w_node_context.project_platform_id > 0
GROUP BY workflow.id)
AND last_execution>NOW()- INTERVAL '5 DAY';
