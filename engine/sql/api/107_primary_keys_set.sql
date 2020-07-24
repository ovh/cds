-- +migrate Up
 
select create_primary_key('application_pipeline_notif', 'application_pipeline_id,environment_id');
select create_primary_key('environment_key', 'ID');
select create_primary_key('project_key', 'ID');
select create_primary_key('user_key', 'user_id,user_key');
select create_primary_key('worker_capability', 'worker_model_id,type,name');
select create_primary_key('workflow_node_run_artifacts', 'ID');
select create_primary_key('pipeline_trigger', 'ID');
select create_primary_key('application_key', 'ID');
select create_primary_key('template_action', 'template_id,action_id');
select create_primary_key('action_audit', 'action_id,user_id,versionned');
select create_primary_key('gorp_migrations_lock', 'ID');

DROP TABLE IF EXISTS sla;

-- +migrate Down

select 1;
