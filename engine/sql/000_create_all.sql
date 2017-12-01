-- +migrate Up
-- +migrate StatementBegin
CREATE OR REPLACE FUNCTION create_index(table_name text, index_name text, column_name text) RETURNS void AS $$
declare
   l_count integer;
begin
  select count(*)
     into l_count
  from pg_indexes
  where schemaname = 'public'
    and tablename = lower(table_name)
    and indexname = lower(index_name);

  if l_count = 0 then
     execute 'create index ' || index_name || ' on "' || table_name || '"(' || column_name || ')';
  end if;
end;
$$ LANGUAGE plpgsql;
-- +migrate StatementEnd

-- +migrate StatementBegin
CREATE OR REPLACE FUNCTION create_unique_index(table_name text, index_name text, column_names text) RETURNS void AS $$
declare
   l_count integer;
begin
  select count(*)
     into l_count
  from pg_indexes
  where schemaname = 'public'
    and tablename = lower(table_name)
    and indexname = lower(index_name);

  if l_count = 0 then
     execute 'create unique index ' || index_name || ' on "' || table_name || '"(' || array_to_string(string_to_array(column_names, ',') , ',') || ')';
  end if;
end;
$$ LANGUAGE plpgsql;
-- +migrate StatementEnd

-- +migrate StatementBegin
CREATE OR REPLACE FUNCTION create_foreign_key(fk_name text, table_name_child text, table_name_parent text, column_name_child text, column_name_parent text) RETURNS void AS $$
declare
   l_count integer;
begin
  select count(*)
     into l_count
  from information_schema.table_constraints as tc
  where constraint_type = 'FOREIGN KEY'
    and tc.table_name = lower(table_name_child)
    and tc.constraint_name = lower(fk_name);

  if l_count = 0 then
     execute 'alter table "' || table_name_child || '" ADD CONSTRAINT ' || fk_name || ' FOREIGN KEY(' || column_name_child || ') REFERENCES "' || table_name_parent || '"(' || column_name_parent || ')';
  end if;
end;
$$ LANGUAGE plpgsql;
-- +migrate StatementEnd

CREATE TABLE IF NOT EXISTS "action" (id BIGSERIAL PRIMARY KEY, name TEXT, type TEXT, description TEXT, enabled BOOLEAN, public BOOLEAN, last_modified TIMESTAMP WITH TIME ZONE DEFAULT  LOCALTIMESTAMP);
CREATE TABLE IF NOT EXISTS "action_requirement" (id BIGSERIAL PRIMARY KEY, action_id BIGINT, name TEXT, type TEXT, value TEXT);
CREATE TABLE IF NOT EXISTS "action_edge" (id BIGSERIAL PRIMARY KEY, parent_id BIGINT, child_id BIGINT, exec_order INT, final boolean not null default false, enabled boolean not null default true);
CREATE TABLE IF NOT EXISTS "action_edge_parameter" (id BIGSERIAL PRIMARY KEY, action_edge_id BIGINT, name TEXT, type TEXT, value TEXT, description TEXT);
CREATE TABLE IF NOT EXISTS "action_parameter" (id BIGSERIAL PRIMARY KEY, action_id BIGINT, name TEXT, type TEXT, value TEXT, description TEXT, worker_model_name TEXT);
CREATE TABLE IF NOT EXISTS "action_build" (id BIGSERIAL PRIMARY KEY, pipeline_action_id INT, args TEXT, status TEXT, pipeline_build_id INT, queued TIMESTAMP WITH TIME ZONE, start TIMESTAMP WITH TIME ZONE, done TIMESTAMP WITH TIME ZONE, worker_model_name TEXT);
CREATE TABLE IF NOT EXISTS "action_audit" (action_id BIGINT, user_id BIGINT, change TEXT, versionned TIMESTAMP WITH TIME ZONE, action_json JSONB);

CREATE TABLE IF NOT EXISTS "artifact" (id BIGSERIAL PRIMARY KEY, name TEXT, tag TEXT, pipeline_id INT, application_id INT, environment_id INT, build_number INT, download_hash TEXT, size BIGINT, perm INT, md5sum TEXT, object_path TEXT, created TIMESTAMP WITH TIME ZONE DEFAULT LOCALTIMESTAMP);

CREATE TABLE IF NOT EXISTS "activity" (day DATE, project_id BIGINT, application_id BIGINT, build BIGINT, unit_test BIGINT, testing BIGINT, deployment BIGINT, PRIMARY KEY(day, project_id, application_id));

CREATE TABLE IF NOT EXISTS "application" (id BIGSERIAL PRIMARY KEY, name TEXT, project_id INT, description TEXT, repo_fullname TEXT, repositories_manager_id BIGINT, last_modified TIMESTAMP WITH TIME ZONE DEFAULT  LOCALTIMESTAMP);
CREATE TABLE IF NOT EXISTS "application_group" (application_id INT, group_id INT, role INT, PRIMARY KEY(group_id, application_id));
CREATE TABLE IF NOT EXISTS "application_pipeline" (id BIGSERIAL PRIMARY KEY, application_id INT, pipeline_id INT, args TEXT, last_modified TIMESTAMP WITH TIME ZONE DEFAULT LOCALTIMESTAMP);
CREATE TABLE IF NOT EXISTS "application_variable" (id BIGSERIAL, application_id INT, var_name TEXT, var_value TEXT, cipher_value BYTEA, var_type TEXT,PRIMARY KEY(application_id, var_name) );
CREATE TABLE IF NOT EXISTS "application_variable_audit" (id BIGSERIAL PRIMARY KEY, application_id BIGINT, data TEXT, author TEXT, versionned TIMESTAMP WITH TIME ZONE);
CREATE TABLE IF NOT EXISTS "application_pipeline_notif" (application_pipeline_id BIGINT, environment_id BIGINT, settings JSONB);

CREATE TABLE IF NOT EXISTS "build_log" (id BIGSERIAL PRIMARY KEY, action_build_id INT, "timestamp" TIMESTAMP WITH TIME ZONE, step TEXT, value TEXT);

CREATE TABLE IF NOT EXISTS "environment" (id BIGSERIAL PRIMARY KEY, name TEXT, project_id INT, created TIMESTAMP WITH TIME ZONE DEFAULT LOCALTIMESTAMP, last_modified TIMESTAMP WITH TIME ZONE DEFAULT LOCALTIMESTAMP);
CREATE TABLE IF NOT EXISTS "environment_variable" (id BIGSERIAL, environment_id INT, name TEXT, value TEXT, cipher_value BYTEA, type TEXT,description TEXT, PRIMARY KEY(environment_id, name) );
CREATE TABLE IF NOT EXISTS "environment_variable_audit" (id BIGSERIAL PRIMARY KEY, environment_id BIGINT, versionned TIMESTAMP WITH TIME ZONE, data TEXT, author TEXT);
CREATE TABLE IF NOT EXISTS "environment_group" (id BIGSERIAL, environment_id INT, group_id INT, role INT, PRIMARY KEY(group_id, environment_id));

CREATE TABLE IF NOT EXISTS "group" (id BIGSERIAL PRIMARY KEY, name TEXT);
CREATE TABLE IF NOT EXISTS "group_user" (id BIGSERIAL, group_id INT, user_id INT, group_admin BOOL, PRIMARY KEY(group_id, user_id));

CREATE TABLE IF NOT EXISTS "hatchery" (id BIGSERIAL PRIMARY KEY, name TEXT, last_beat TIMESTAMP WITH TIME ZONE, uid TEXT, group_id INT, status TEXT);
CREATE TABLE IF NOT EXISTS "hatchery_model" (hatchery_id BIGINT, worker_model_id BIGINT, PRIMARY KEY(hatchery_id, worker_model_id));
CREATE TABLE IF NOT EXISTS "hook" (id BIGSERIAL PRIMARY KEY, pipeline_id BIGINT, application_id INT,  kind TEXT, host TEXT, project TEXT, repository TEXT, uid TEXT, enabled BOOL);

CREATE TABLE IF NOT EXISTS "pipeline" (id BIGSERIAL PRIMARY KEY, name TEXT, project_id INT, type TEXT, created TIMESTAMP WITH TIME ZONE DEFAULT LOCALTIMESTAMP, last_modified TIMESTAMP WITH TIME ZONE DEFAULT  LOCALTIMESTAMP);
CREATE TABLE IF NOT EXISTS "pipeline_action" (id BIGSERIAL PRIMARY KEY, pipeline_stage_id INT, action_id INT, args TEXT, enabled BOOLEAN, last_modified TIMESTAMP WITH TIME ZONE DEFAULT  LOCALTIMESTAMP);
CREATE TABLE IF NOT EXISTS "pipeline_build" (id BIGSERIAL PRIMARY KEY, environment_id INT, application_id INT, pipeline_id INT, build_number INT, version BIGINT, status TEXT, args TEXT, start TIMESTAMP WITH TIME ZONE, done TIMESTAMP WITH TIME ZONE, manual_trigger BOOLEAN, triggered_by BIGINT, parent_pipeline_build_id BIGINT, vcs_changes_branch TEXT, vcs_changes_hash TEXT, vcs_changes_author TEXT, scheduled_trigger BOOLEAN default FALSE);
CREATE TABLE IF NOT EXISTS "pipeline_build_test" (pipeline_build_id BIGINT PRIMARY KEY, tests TEXT);
CREATE TABLE IF NOT EXISTS "pipeline_group" (id BIGSERIAL, pipeline_id INT, group_id INT, role INT, PRIMARY KEY(group_id, pipeline_id));
CREATE TABLE IF NOT EXISTS "pipeline_history" (pipeline_build_id BIGINT, pipeline_id INT, application_id INT, environment_id INT, build_number INT, version BIGINT, status TEXT, start TIMESTAMP WITH TIME ZONE, done TIMESTAMP WITH TIME ZONE, data json, manual_trigger BOOLEAN, triggered_by BIGINT, parent_pipeline_build_id BIGINT, vcs_changes_branch TEXT, vcs_changes_hash TEXT, vcs_changes_author TEXT, scheduled_trigger BOOLEAN default FALSE, PRIMARY KEY(pipeline_id, application_id, build_number, environment_id));
CREATE TABLE IF NOT EXISTS "pipeline_stage" (id BIGSERIAL PRIMARY KEY, pipeline_id INT, name TEXT, build_order INT, enabled BOOLEAN, last_modified TIMESTAMP WITH TIME ZONE DEFAULT  LOCALTIMESTAMP);
CREATE TABLE IF NOT EXISTS "pipeline_stage_prerequisite" (id BIGSERIAL PRIMARY KEY, pipeline_stage_id BIGINT, parameter TEXT, expected_value TEXT);
CREATE TABLE IF NOT EXISTS "pipeline_parameter" (id BIGSERIAL, pipeline_id INT, name TEXT, value TEXT, type TEXT,description TEXT, PRIMARY KEY(pipeline_id, name));

CREATE TABLE IF NOT EXISTS "pipeline_scheduler" (id BIGSERIAL PRIMARY KEY, application_id BIGINT NOT NULL, pipeline_id BIGINT NOT NULL, environment_id BIGINT NOT NULL, args JSONB, crontab TEXT NOT NULL, disable BOOLEAN DEFAULT FALSE);
CREATE TABLE IF NOT EXISTS "pipeline_scheduler_execution" (id BIGSERIAL PRIMARY KEY, pipeline_scheduler_id BIGINT NOT NULL, execution_planned_date TIMESTAMP WITH TIME ZONE, execution_date TIMESTAMP WITH TIME ZONE, executed BOOLEAN NOT NULL DEFAULT FALSE, pipeline_build_version BIGINT);

CREATE TABLE IF NOT EXISTS "pipeline_trigger" (id BIGSERIAL PRIMARY KEY, src_application_id INT, src_pipeline_id INT, src_environment_id INT, dest_application_id INT, dest_pipeline_id INT, dest_environment_id INT, manual BOOL, last_modified TIMESTAMP WITH TIME ZONE DEFAULT  LOCALTIMESTAMP);
CREATE TABLE IF NOT EXISTS "pipeline_trigger_parameter" (id BIGSERIAL PRIMARY KEY, pipeline_trigger_id BIGINT, name TEXT, type TEXT, value TEXT, description TEXT);

CREATE TABLE IF NOT EXISTS "pipeline_trigger_prerequisite" (id BIGSERIAL PRIMARY KEY, pipeline_trigger_id BIGINT, parameter TEXT, expected_value TEXT);

CREATE TABLE IF NOT EXISTS "plugin" (id BIGSERIAL PRIMARY KEY, name TEXT, size BIGINT, perm INT, md5sum TEXT, object_path TEXT);

CREATE TABLE IF NOT EXISTS "poller" (application_id BIGINT, pipeline_id BIGINT, enabled BOOLEAN, name TEXT, date_creation TIMESTAMP WITH TIME ZONE, PRIMARY KEY(application_id, pipeline_id));
CREATE TABLE IF NOT EXISTS "poller_execution" (id BIGSERIAL PRIMARY KEY, application_id BIGINT, pipeline_id BIGINT, execution_date TIMESTAMP WITH TIME ZONE, status TEXT, data JSONB);

CREATE TABLE IF NOT EXISTS "project" (id BIGSERIAL PRIMARY KEY, projectKey TEXT , name TEXT, created TIMESTAMP WITH TIME ZONE DEFAULT LOCALTIMESTAMP, last_modified TIMESTAMP WITH TIME ZONE DEFAULT  LOCALTIMESTAMP);
CREATE TABLE IF NOT EXISTS "project_group" (id BIGSERIAL, project_id INT, group_id INT, role INT,PRIMARY KEY(group_id, project_id));
CREATE TABLE IF NOT EXISTS "project_variable" (id BIGSERIAL, project_id INT, var_name TEXT, var_value TEXT, cipher_value BYTEA, var_type TEXT,PRIMARY KEY(project_id, var_name));
CREATE TABLE IF NOT EXISTS "project_variable_audit" (id BIGSERIAL PRIMARY KEY, project_id BIGINT, versionned TIMESTAMP WITH TIME ZONE, data TEXT, author TEXT);

CREATE TABLE IF NOT EXISTS "received_hook" (id BIGSERIAL PRIMARY KEY, link TEXT, data TEXT);

CREATE TABLE IF NOT EXISTS "repositories_manager" (id BIGSERIAL PRIMARY KEY , type TEXT, name TEXT UNIQUE, url TEXT UNIQUE, data JSONB );
CREATE TABLE IF NOT EXISTS "repositories_manager_project" ( id_repositories_manager BIGINT NOT NULL, id_project BIGINT NOT NULL, data JSONB, PRIMARY KEY(id_repositories_manager, id_project));

CREATE TABLE IF NOT EXISTS "stats" (day DATE PRIMARY KEY, build BIGINT, unit_test BIGINT, testing BIGINT, deployment BIGINT, max_building_worker BIGINT, max_building_pipeline BIGINT);

CREATE TABLE IF NOT EXISTS "system_log" (id BIGSERIAL PRIMARY KEY, logged TIMESTAMP WITH TIME ZONE, level TEXT, log TEXT);

CREATE TABLE IF NOT EXISTS "template" (id BIGSERIAL PRIMARY KEY, name TEXT, type TEXT,author TEXT, description TEXT, identifier TEXT, size INTEGER, perm INTEGER, md5sum TEXT, object_path TEXT);
CREATE TABLE IF NOT EXISTS "template_params" (template_id BIGINT NOT NULL, params JSONB);
CREATE TABLE IF NOT EXISTS "template_action" (template_id BIGINT NOT NULL, action_id BIGINT NOT NULL);
CREATE TABLE IF NOT EXISTS "token" (group_id INT, token TEXT, expiration INT, created TIMESTAMP WITH TIME ZONE);

CREATE TABLE IF NOT EXISTS "user" (id BIGSERIAL PRIMARY KEY, username TEXT, admin BOOL, data TEXT, auth TEXT, created TIMESTAMP WITH TIME ZONE, origin TEXT);
CREATE TABLE IF NOT EXISTS "user_key" (user_id INT, user_key TEXT, expiry INT DEFAULT 0);
CREATE TABLE IF NOT EXISTS "user_notification" (id BIGSERIAL PRIMARY KEY, type TEXT, content JSONB, status TEXT, creation_date INT);

CREATE TABLE IF NOT EXISTS "warning" (id BIGSERIAL PRIMARY KEY, project_id BIGINT, app_id BIGINT, pip_id BIGINT, env_id BIGINT, action_id BIGINT, warning_id BIGINT, message_param JSONB);

CREATE TABLE IF NOT EXISTS "worker" (id TEXT PRIMARY KEY, name TEXT, last_beat TIMESTAMP WITH TIME ZONE, owner_id INT, group_id INT, model INT, status TEXT, action_build_id BIGINT, hatchery_id BIGINT DEFAULT 0);
CREATE TABLE IF NOT EXISTS "worker_capability" (worker_model_id INT, type TEXT, name TEXT, argument TEXT);
CREATE TABLE IF NOT EXISTS "worker_model" (id BIGSERIAL PRIMARY KEY, type TEXT, name TEXT, image TEXT, created_by JSONB, GROUP_ID BIGINT, OWNER_ID BIGINT);

-- ACTION REQUIREMENT
select create_index('action_requirement', 'IDX_ACTION_REQUIREMENT_ACTION_ID', 'action_id');

-- ACTION_EDGE
select create_index('action_edge', 'IDX_ACTION_EDGE_PARENT_ID', 'parent_id');
select create_index('action_edge', 'IDX_ACTION_EDGE_CHILD_ID', 'child_id');

-- ACTION EDGE PARAMETER
select create_index('action_edge_parameter', 'IDX_ACTION_EDGE_PARAMETER_ACTION_EDGE_ID', 'action_edge_id');

-- ACTION PARAMETER
select create_unique_index('action_parameter', 'IDX_ACTION_PARAMETER_ACTION_ID', 'action_id,name');

-- PIPELINE BUILD
select create_index('pipeline_build', 'IDX_PIPELINE_BUILD_UNIQUE_BUILD_NUMBER', 'build_number,pipeline_id,application_id,environment_id');

-- ACTION BUILD
select create_index('action_build', 'IDX_ACTION_BUILD_PIPELINE_BUILD_ID', 'pipeline_build_id');
select create_index('action_build', 'IDX_ACTION_BUILD_PIPELINE_ACTION_ID', 'pipeline_action_id');
select create_unique_index('action_build', 'IDX_ACTION_BUILD_PIPELINE_ACTION_ID_BUILD_ID', 'pipeline_build_id,pipeline_action_id');

-- ARTIFACT
select create_index('artifact', 'IDX_ARTIFACT_PIPELINE_ID', 'pipeline_id');
select create_index('artifact', 'IDX_ARTIFACT_APPLICATION_ID', 'application_id');
select create_index('artifact','IDX_ARTIFACT_ENVIRONMENT', 'environment_id');

-- APPLICATION
select create_unique_index('application', 'IDX_APPLICATION_PROJECT_ID_NAME', 'project_id,name');

-- APPLICATION PIPELINE
select create_unique_index('application_pipeline','IDX_APPLICATION_PIPELINE_APPLICATION', 'application_id,pipeline_id');

-- BUILD_LOG
select create_index('build_log', 'IDX_ARTIFACT_ACTION_BUILD_ID', 'action_build_id');

-- ENVIRONMENT
select create_unique_index('environment','IDX_ENVIRONMENT', 'name,project_id');

-- ENVIRONMENT_GROUP
select create_unique_index('environment_group','IDX_ENVIRONMENT_GROUP_UNIQUE', 'group_id,environment_id');
select create_index('environment_group','IDX_ENVIRONMENT_GROUP_GROUP', 'group_id');
select create_index('environment_group','IDX_ENVIRONMENT_GROUP_ENV', 'environment_id');

-- ENVIRONMENT_VARIABLE
select create_unique_index('environment_variable','IDX_ENVIRONMENT_VARIABLE_ID_NAME', 'environment_id,name');

-- GROUP
select create_unique_index('group', 'IDX_GROUP_NAME', 'name');

-- HOOK
select create_index('hook','IDX_HOOK_PIPELINE_ID','pipeline_id');

-- PIPELINE
select create_unique_index('pipeline','IDX_PIPELINE_NAME','name,project_id');
select create_index('pipeline','IDX_PIPELINE_PROJECT_ID','project_id');

-- PIPELINE_ACTION
select create_index('pipeline_action','IDX_PIPELINE_ACTION_STAGE_ID','pipeline_stage_id');
select create_index('pipeline_action','IDX_PIPELINE_ACTION_ACTION_ID','action_id');

-- PIPELINE BUILD
select create_index('pipeline_build','IDX_PIPELINE_BUILD_PIPELINE_ID','pipeline_id');
select create_index('pipeline_build','IDX_PIPELINE_BUILD_NUMBER','build_number');
select create_index('pipeline_build','IDX_PIPELINE_BUILD_APPLICATION_ID','application_id');
select create_index('pipeline_build','IDX_PIPELINE_BUILD_ENVIRONMENT_ID','environment_id');

-- PIPELINE PARAMETER
select create_unique_index('pipeline_parameter','IDX_PIPELINE_PARAMETER_NAME','pipeline_id,name');

-- PIPELINE HISTORY
select create_index('pipeline_history', 'IDX_PIPELINE_HISTORY', 'build_number');
select create_index('pipeline_history','IDX_PIPELINE_HISTORY_ENVIRONMENT', 'environment_id');

-- PIPELINE Stage
select create_index('pipeline_stage','IDX_PIPELINE_STAGE_BUILD_ORDER','build_order');
select create_index('pipeline_stage','IDX_PIPELINE_STAGE_PIPELINE_ID','pipeline_id');

-- PIPELINE TRIGGER
select create_index('pipeline_trigger','IDX_PIPELINE_TRIGGER_SRC_APPLICATION', 'src_application_id');
select create_index('pipeline_trigger','IDX_PIPELINE_TRIGGER_DEST_APPLICATION', 'dest_application_id');
select create_index('pipeline_trigger','IDX_PIPELINE_TRIGGER_SRC_PIPELINE', 'src_pipeline_id');
select create_index('pipeline_trigger','IDX_PIPELINE_TRIGGER_DEST_PIPELINE', 'dest_pipeline_id');
select create_index('pipeline_trigger','IDX_PIPELINE_TRIGGER_SRC_ENVIRONMENT', 'src_environment_id');
select create_index('pipeline_trigger','IDX_PIPELINE_TRIGGER_DEST_ENVIRONMENT', 'dest_environment_id');

-- PLUGIN
select create_unique_index('plugin','IDX_PLUGIN_NAME', 'name');

-- PROJECT
select create_unique_index('project','IDX_PROJECT_KEY','projectKey');

-- SYSTEM_LOG
select create_index('system_log','IDX_SYS_LOG_LOGGED','logged');

-- USER
select create_unique_index('user','IDX_USER_USERNAME','username');

-- USER KEY
select create_index('user_key','IDX_USER_KEY_USER_KEY','user_key');

-- WORKER
select create_index('worker','IDX_WORKER_ID','id');

-- WORKER_CAPABILITY
select create_unique_index('worker_capability','IDX_WORKER_CAPABILITY_NAME','worker_model_id,name');
select create_index('worker_capability','IDX_WORKER_CAPABILITY_MODEL_ID','worker_model_id');

-- WORKER_MODEL
select create_unique_index('worker_model','IDX_WORKER_MODEL_NAME','name');
select create_index('worker_model','IDX_WORKER_MODEL_GROUP_ID','group_id');

-- REPOSITORIES_MANAGER_PROJECT
select create_unique_index('repositories_manager_project', 'IDX_REPOSITORIES_MANAGER_PROJECT_ID' ,'id_repositories_manager, id_project');

-- TEMPLATES
SELECT create_unique_index('template', 'IDX_TEMPLATE_IDENTIFIER', 'identifier');
SELECT create_unique_index('template', 'IDX_TEMPLATE_NAME', 'name');

-- ACTION BUILD
select create_foreign_key('FK_ACTION_BUILD_PIPELINE_ACTION', 'action_build', 'pipeline_action', 'pipeline_action_id', 'id');
select create_foreign_key('FK_ACTION_BUILD_PIPELINE_BUILD', 'action_build', 'pipeline_build', 'pipeline_build_id', 'id');

-- ACTION REQUIREMENT
select create_foreign_key('FK_ACTION_REQUIREMENT_ACTION', 'action_requirement', 'action', 'action_id', 'id');

-- ACTION EDGE
select create_foreign_key('FK_ACTION_EDGE_PARENT_ACTION', 'action_edge', 'action', 'parent_id', 'id');
select create_foreign_key('FK_ACTION_EDGE_CHILD_ACTION', 'action_edge', 'action', 'child_id', 'id');

-- ACTION EDGE PARAMETER
select create_foreign_key('FK_ACTION_EDGE_PARAMETER_ACTION_EDGE', 'action_edge_parameter', 'action_edge', 'action_edge_id', 'id');

-- ACTION PARAMETER
select create_foreign_key('FK_ACTION_PARAMETER_ACTION', 'action_parameter', 'action', 'action_id', 'id');

-- ARTIFACT
select create_foreign_key('FK_ARTIFACT_PIPELINE_BUILD', 'artifact', 'pipeline', 'pipeline_id', 'id');
select create_foreign_key('FK_ARTIFACT_APPLICATION', 'artifact', 'application', 'application_id', 'id');
select create_foreign_key('FK_ARTIFACT_ENVIRONMENT', 'artifact', 'environment', 'environment_id', 'id');

-- APPLICATION
select create_foreign_key('FK_APPLICATION_PROJECT', 'application', 'project', 'project_id', 'id');
select create_foreign_key('FK_APPLICATION_REPOSITORIES_MANAGER', 'application', 'repositories_manager', 'repositories_manager_id', 'id');

-- APPLICATION GROUP
select create_foreign_key('FK_APPLICATION_GROUP_APPLICATION', 'application_group', 'application', 'application_id', 'id');
select create_foreign_key('FK_APPLICATION_GROUP_GROUP', 'application_group', 'group', 'group_id', 'id');

-- APPLICATION PIPELINE
select create_foreign_key('FK_APPLICATION_PIPELINE_APPLICATION', 'application_pipeline', 'application', 'application_id', 'id');
select create_foreign_key('FK_APPLICATION_PIPELINE_PIPELINE', 'application_pipeline', 'pipeline', 'pipeline_id', 'id');

-- APPLICATION_VARIABLE
select create_foreign_key('FK_APPLICATION_VARIABLE_APPLICATION', 'application_variable', 'application', 'application_id', 'id');

-- APPLICATION PIPELINE NOTIF
SELECT create_foreign_key('FK_APPLICATION_PIPELINE_NOTIF_APPLICATION_PIPELINE', 'application_pipeline_notif', 'application_pipeline', 'application_pipeline_id', 'id');
SELECT create_foreign_key('FK_APPLICATION_PIPELINE_NOTIF_ENVIRONMENT', 'application_pipeline_notif', 'environment', 'environment_id', 'id');

-- BUILD_LOG
select create_foreign_key('FK_BUILD_LOG_ACTION_BUILD', 'build_log', 'action_build', 'action_build_id', 'id');

-- ENVIRONMENT
select create_foreign_key('FK_ENVIRONMENT_PROJECT', 'environment', 'project', 'project_id', 'id');

-- ENVIRONMENT_GROUP
select create_foreign_key('FK_ENVIRONMENT_GROUP_ENV', 'environment_group', 'environment', 'environment_id', 'id');
select create_foreign_key('FK_ENVIRONMENT_GROUP_GROUP', 'environment_group', 'group', 'group_id', 'id');

-- ENVIRONMENT_VARIABLE
select create_foreign_key('FK_ENVIRONMENT_VARIABLE_ENV', 'environment_variable', 'environment', 'environment_id', 'id');

-- GROUP USER
select create_foreign_key('FK_GROUP_USER_GROUP', 'group_user', 'group', 'group_id', 'id');
select create_foreign_key('FK_GROUP_USER_USER', 'group_user', 'user', 'user_id', 'id');

-- HOOK
select create_foreign_key('FK_HOOK_PIPELINE', 'hook', 'pipeline', 'pipeline_id', 'id');
select create_foreign_key('FK_HOOK_APPLICATION', 'hook', 'application', 'application_id', 'id');

-- PIPELINE
select create_foreign_key('FK_PIPELINE_PROJECT', 'pipeline', 'project', 'project_id', 'id');

-- PIPELINE ACTION
select create_foreign_key('FK_PIPELINE_ACTION_ACTION', 'pipeline_action', 'action', 'action_id', 'id');
select create_foreign_key('FK_PIPELINE_ACTION_PIPELINE_STAGE', 'pipeline_action', 'pipeline_stage', 'pipeline_stage_id', 'id');

-- PIPELINE BUILD
select create_foreign_key('FK_PIPELINE_BUILD_PIPELINE', 'pipeline_build', 'pipeline', 'pipeline_id', 'id');
select create_foreign_key('FK_PIPELINE_BUILD_APPLICATION', 'pipeline_build', 'application', 'application_id', 'id');
select create_foreign_key('FK_PIPELINE_BUILD_ENVIRONMENT', 'pipeline_build', 'environment', 'environment_id', 'id');

-- PIPELINE GROUP
select create_foreign_key('FK_PIPELINE_GROUP_PIPELINE', 'pipeline_group', 'pipeline', 'pipeline_id', 'id');
select create_foreign_key('FK_PIPELINE_GROUP', 'pipeline_group', 'group', 'group_id', 'id');

-- PIPELINE HISTORY
select create_foreign_key('FK_PIPELINE_HISTORY_PIPELINE', 'pipeline_history', 'pipeline', 'pipeline_id', 'id');
select create_foreign_key('FK_PIPELINE_HISTORY_ENVIRONMENT', 'pipeline_history', 'environment', 'environment_id', 'id');

-- PIPELINE STAGE
select create_foreign_key('FK_PIPELINE_STAGE_PIPELINE', 'pipeline_stage', 'pipeline', 'pipeline_id', 'id');

-- PIPELINE STAGE PREREQUISITE
select create_foreign_key('FK_PIPELINE_STAGE_PREREQUISITE_PIPELINE_STAGE', 'pipeline_stage_prerequisite', 'pipeline_stage', 'pipeline_stage_id', 'id');

-- PIPELINE PARAMETER
select create_foreign_key('FK_PIPELINE_PARAMETER_PIPELINE', 'pipeline_parameter', 'pipeline', 'pipeline_id', 'id');

-- PIPELINE SCHEDULER
SELECT create_foreign_key('FK_PIPELINE_SCHEDULER_APPLICATION', 'pipeline_scheduler', 'application', 'application_id', 'id');
SELECT create_foreign_key('FK_PIPELINE_SCHEDULER_PIPELINE', 'pipeline_scheduler', 'pipeline', 'pipeline_id', 'id');
SELECT create_foreign_key('FK_PIPELINE_SCHEDULER_ENVIRONMENT', 'pipeline_scheduler', 'environment', 'environment_id', 'id');
SELECT create_foreign_key('FK_PIPELINE_SCHEDULER_EXECUTION_PIPELINE_SCHEDULER', 'pipeline_scheduler_execution', 'pipeline_scheduler', 'pipeline_scheduler_id', 'id');


-- PIPELINE TRIGGER
select create_foreign_key('FK_PIPELINE_TRIGGER_SRC_APPLICATION', 'pipeline_trigger', 'application', 'src_application_id', 'id');
select create_foreign_key('FK_PIPELINE_TRIGGER_DEST_APPLICATION', 'pipeline_trigger', 'application', 'dest_application_id', 'id');
select create_foreign_key('FK_PIPELINE_TRIGGER_SRC_PIPELINE', 'pipeline_trigger', 'pipeline', 'src_pipeline_id', 'id');
select create_foreign_key('FK_PIPELINE_TRIGGER_DEST_PIPELINE', 'pipeline_trigger', 'pipeline', 'dest_pipeline_id', 'id');
select create_foreign_key('FK_PIPELINE_TRIGGER_SRC_ENVIRONMENT', 'pipeline_trigger', 'environment', 'src_environment_id', 'id');
select create_foreign_key('FK_PIPELINE_TRIGGER_DEST_ENVIRONMENT', 'pipeline_trigger', 'environment', 'dest_environment_id', 'id');

-- PIPELINE TRIGGER PARAMETER
select create_foreign_key('FK_PIPELINE_TRIGGER_PARAMETER_PIPELINE', 'pipeline_trigger_parameter', 'pipeline_trigger', 'pipeline_trigger_id', 'id');

-- PIPELINE TRIGGER PREREQUISITE
select create_foreign_key('FK_PIPELINE_TRIGGER_PREREQUISITE_PIPELINE_TRIGGER', 'pipeline_trigger_prerequisite', 'pipeline_trigger', 'pipeline_trigger_id', 'id');

-- POLLER
select create_foreign_key('FK_POLLER_APPLICATION', 'poller', 'application', 'application_id', 'id');
select create_foreign_key('FK_POLLER_PIPELINE', 'poller', 'pipeline', 'pipeline_id', 'id');

-- POLLER EXECUTION
select create_foreign_key('FK_POLLER_EXECUTION_APPLICATION', 'poller_execution', 'application', 'application_id', 'id');
select create_foreign_key('FK_POLLER_EXECUTION_PIPELINE', 'poller_execution', 'pipeline', 'pipeline_id', 'id');

-- PROJECT GROUP
select create_foreign_key('FK_PROJECT_GROUP_PIPELINE', 'project_group', 'project', 'project_id', 'id');
select create_foreign_key('FK_PROJECT_GROUP', 'project_group', 'group', 'group_id', 'id');

-- PROJECT VARIABLE
select create_foreign_key('FK_PROJECT_VARIABLE_PIPELINE', 'project_variable', 'project', 'project_id', 'id');

-- USER KEY
select create_foreign_key('FK_USER_KEY_USER', 'user_key', 'user', 'user_id', 'id');

-- WORKER CAPABILITY
select create_foreign_key('FK_WORKER_CAPABILITY_WORKER_MODEL', 'worker_capability', 'worker_model', 'worker_model_id', 'id');

-- WORKER
select create_foreign_key('FK_WORKER_ACTION_BUILD', 'worker', 'action_build', 'action_build_id', 'id');

-- WORKER MODEL
select create_foreign_key('FK_WORKER_MODEL_GROUP', 'worker_model', 'group', 'group_id', 'id');

-- HATCHERY MODEL
select create_foreign_key('FK_HATCHERY_MODEL_HATCHERY_ID', 'hatchery_model', 'hatchery', 'hatchery_id', 'id');
select create_foreign_key('FK_HATCHERY_MODEL_WORKER_MODEL_ID', 'hatchery_model', 'worker_model', 'worker_model_id', 'id');

-- repositories_manager_project
select create_foreign_key('fk_repositories_manager_project_repositories_manager_id', 'repositories_manager_project', 'repositories_manager', 'id_repositories_manager', 'id');
select create_foreign_key('fk_repositories_manager_project_project_id', 'repositories_manager_project', 'project', 'id_project', 'id');

-- warning
ALTER TABLE warning ADD CONSTRAINT fk_application FOREIGN KEY (app_id) references application (id) ON delete cascade;
ALTER TABLE warning ADD CONSTRAINT fk_pipeline FOREIGN KEY (pip_id) references pipeline (id) ON delete cascade;
ALTER TABLE warning ADD CONSTRAINT fk_environment FOREIGN KEY (env_id) references environment (id) ON delete cascade;
ALTER TABLE warning ADD CONSTRAINT fk_action FOREIGN KEY (action_id) references action (id) ON delete cascade;

-- AUDIT
ALTER TABLE project_variable_audit ADD CONSTRAINT fk_project FOREIGN KEY (project_id) references project (id) ON delete cascade;
ALTER TABLE application_variable_audit ADD CONSTRAINT fk_application FOREIGN KEY (application_id) references application (id) ON delete cascade;
ALTER TABLE environment_variable_audit ADD CONSTRAINT fk_environment FOREIGN KEY (environment_id) references environment (id) ON delete cascade;

-- TEMPLATES
SELECT create_foreign_key('FK_TEMPLATE_PARAMS_TEMPLATE', 'template_params', 'template', 'template_id', 'id');
SELECT create_foreign_key('FK_TEMPLATE_ACTIONS_TEMPLATE', 'template_params', 'template', 'template_id', 'id');
SELECT create_foreign_key('FK_TEMPLATE_ACTIONS_ACTION', 'template_action', 'action', 'action_id', 'id');

-- +migrate Down
-- nothing to downgrade, it's a creation !