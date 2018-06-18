 -- +migrate Up
 
-- +migrate StatementBegin
CREATE OR REPLACE FUNCTION create_primary_key(tablename text, column_names text) RETURNS void AS $$
declare
   l_count integer;
begin
  select count(*) into l_count from information_schema.table_constraints where table_name = lower(tablename) and constraint_type = 'PRIMARY KEY';
  if l_count = 0 then
     execute 'ALTER TABLE ' || tablename || ' ADD PRIMARY KEY (' || array_to_string(string_to_array(column_names, ',') , ',') || ')';
  end if;
end;
$$ LANGUAGE plpgsql;
-- +migrate StatementEnd

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

-- +migrate Down

select 1;
