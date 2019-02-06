-- +migrate Up
ALTER TABLE integration_model RENAME COLUMN file_storage TO storage;
ALTER TABLE integration_model DROP COLUMN block_storage;

INSERT into action_parameter (action_id, name, description, type, value, advanced) values((select id from action where name = 'Artifact Upload' and type = 'Builtin'), 'destination', 'Destination of this artifact. Use the name of integration attached on your project', 'string', '', true);
INSERT into action_parameter (action_id, name, description, type, value, advanced) values((select id from action where name = 'Serve Static Files' and type = 'Builtin'), 'destination', 'Destination of uploading. Use the name of integration attached on your project', 'string', '', true);

ALTER TABLE workflow_node_run_artifacts ADD COLUMN project_integration_id BIGINT;
SELECT create_foreign_key('FK_W_NODE_RUN_ARTIFACTS_PRJ_INTEGRATION', 'workflow_node_run_artifacts', 'project_integration', 'project_integration_id', 'id');

ALTER TABLE workflow_node_run_static_files ADD COLUMN project_integration_id BIGINT;
SELECT create_foreign_key('FK_W_NODE_RUN_STATICF_PRJ_INTEGRATION', 'workflow_node_run_static_files', 'project_integration', 'project_integration_id', 'id');

-- +migrate Down

DELETE from action_parameter where name = 'destination' and action_id = (select id from action where name = 'Artifact Upload' and type = 'Builtin');
DELETE from action_parameter where name = 'destination' and action_id = (select id from action where name = 'Serve Static Files' and type = 'Builtin');

ALTER TABLE integration_model RENAME COLUMN storage TO file_storage;
ALTER TABLE integration_model ADD COLUMN block_storage BOOLEAN default false;
ALTER TABLE workflow_node_run_static_files DROP COLUMN project_integration_id;
ALTER TABLE workflow_node_run_artifacts DROP COLUMN project_integration_id;
