CREATE TABLE IF NOT EXISTS "action" (id BIGSERIAL PRIMARY KEY, name TEXT, type TEXT, description TEXT, enabled BOOLEAN, public BOOLEAN, last_modified TIMESTAMP WITH TIME ZONE DEFAULT  LOCALTIMESTAMP);
CREATE TABLE IF NOT EXISTS "action_requirement" (id BIGSERIAL PRIMARY KEY, action_id BIGINT, name TEXT, type TEXT, value TEXT);
CREATE TABLE IF NOT EXISTS "action_edge" (id BIGSERIAL PRIMARY KEY, parent_id BIGINT, child_id BIGINT, exec_order INT, final boolean not null default false, enabled boolean not null default true);
CREATE TABLE IF NOT EXISTS "action_edge_parameter" (id BIGSERIAL PRIMARY KEY, action_edge_id BIGINT, name TEXT, type TEXT, value TEXT, description TEXT);
CREATE TABLE IF NOT EXISTS "action_parameter" (id BIGSERIAL PRIMARY KEY, action_id BIGINT, name TEXT, type TEXT, value TEXT, description TEXT, worker_model_name TEXT);
CREATE TABLE IF NOT EXISTS "action_build" (id BIGSERIAL PRIMARY KEY, pipeline_action_id INT, args TEXT, status TEXT, pipeline_build_id INT, queued TIMESTAMP WITH TIME ZONE, start TIMESTAMP WITH TIME ZONE, done TIMESTAMP WITH TIME ZONE, worker_model_name TEXT);
CREATE TABLE IF NOT EXISTS "action_audit" (action_id BIGINT, user_id BIGINT, change TEXT, versionned TIMESTAMP WITH TIME ZONE, action_json JSONB);

CREATE TABLE IF NOT EXISTS "artifact" (id BIGSERIAL PRIMARY KEY, name TEXT, tag TEXT, pipeline_id INT, application_id INT, environment_id INT, build_number INT, download_hash TEXT, size BIGINT, perm INT, md5sum TEXT, object_path TEXT, created TIMESTAMP WITH TIME ZONE DEFAULT LOCALTIMESTAMP);

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
CREATE TABLE IF NOT EXISTS "hook" (id BIGSERIAL PRIMARY KEY, pipeline_id BIGINT, application_id INT,  kind TEXT, host TEXT, project TEXT, repository TEXT, uid TEXT, enabled BOOL);
CREATE TABLE IF NOT EXISTS "pipeline" (id BIGSERIAL PRIMARY KEY, name TEXT, project_id INT, type TEXT, created TIMESTAMP WITH TIME ZONE DEFAULT LOCALTIMESTAMP, last_modified TIMESTAMP WITH TIME ZONE DEFAULT  LOCALTIMESTAMP);
CREATE TABLE IF NOT EXISTS "pipeline_action" (id BIGSERIAL PRIMARY KEY, pipeline_stage_id INT, action_id INT, args TEXT, enabled BOOLEAN, last_modified TIMESTAMP WITH TIME ZONE DEFAULT  LOCALTIMESTAMP);
CREATE TABLE IF NOT EXISTS "pipeline_build" (id BIGSERIAL PRIMARY KEY, environment_id INT, application_id INT, pipeline_id INT, build_number INT, version BIGINT, status TEXT, args TEXT, start TIMESTAMP WITH TIME ZONE, done TIMESTAMP WITH TIME ZONE, manual_trigger BOOLEAN, triggered_by BIGINT, parent_pipeline_build_id BIGINT, vcs_changes_branch TEXT, vcs_changes_hash TEXT, vcs_changes_author TEXT);
CREATE TABLE IF NOT EXISTS "pipeline_build_test" (pipeline_build_id BIGINT PRIMARY KEY, tests TEXT);

CREATE TABLE IF NOT EXISTS "pipeline_group" (id BIGSERIAL, pipeline_id INT, group_id INT, role INT, PRIMARY KEY(group_id, pipeline_id));
CREATE TABLE IF NOT EXISTS "pipeline_history" (pipeline_build_id BIGINT, pipeline_id INT, application_id INT, environment_id INT, build_number INT, version BIGINT, status TEXT, start TIMESTAMP WITH TIME ZONE, done TIMESTAMP WITH TIME ZONE, data json, manual_trigger BOOLEAN, triggered_by BIGINT, parent_pipeline_build_id BIGINT, vcs_changes_branch TEXT, vcs_changes_hash TEXT, vcs_changes_author TEXT, PRIMARY KEY(pipeline_id, application_id, build_number, environment_id));
CREATE TABLE IF NOT EXISTS "pipeline_stage" (id BIGSERIAL PRIMARY KEY, pipeline_id INT, name TEXT, build_order INT, enabled BOOLEAN, last_modified TIMESTAMP WITH TIME ZONE DEFAULT  LOCALTIMESTAMP);
CREATE TABLE IF NOT EXISTS "pipeline_stage_prerequisite" (id BIGSERIAL PRIMARY KEY, pipeline_stage_id BIGINT, parameter TEXT, expected_value TEXT);
CREATE TABLE IF NOT EXISTS "pipeline_parameter" (id BIGSERIAL, pipeline_id INT, name TEXT, value TEXT, type TEXT,description TEXT, PRIMARY KEY(pipeline_id, name));

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
CREATE TABLE IF NOT EXISTS "system_log" (id BIGSERIAL PRIMARY KEY, logged TIMESTAMP WITH TIME ZONE, level TEXT, log TEXT);
CREATE TABLE IF NOT EXISTS "user" (id BIGSERIAL PRIMARY KEY, username TEXT, admin BOOL, data TEXT, auth TEXT, created TIMESTAMP WITH TIME ZONE, origin TEXT);

CREATE TABLE IF NOT EXISTS "user_key" (user_id INT, user_key TEXT, expiry INT DEFAULT 0);
CREATE TABLE IF NOT EXISTS "token" (group_id INT, token TEXT, expiration INT, created TIMESTAMP WITH TIME ZONE);

CREATE TABLE IF NOT EXISTS "user_notification" (id BIGSERIAL PRIMARY KEY, type TEXT, content JSONB, status TEXT, creation_date INT);

CREATE TABLE IF NOT EXISTS "worker" (id TEXT PRIMARY KEY, name TEXT, last_beat TIMESTAMP WITH TIME ZONE, owner_id INT, group_id INT, model INT, status TEXT, action_build_id BIGINT, hatchery_id BIGINT DEFAULT 0);
CREATE TABLE IF NOT EXISTS "worker_capability" (worker_model_id INT, type TEXT, name TEXT, argument TEXT);
CREATE TABLE IF NOT EXISTS "worker_model" (id BIGSERIAL PRIMARY KEY, type TEXT, name TEXT, image TEXT, owner_id INT);

CREATE TABLE IF NOT EXISTS "hatchery" (id BIGSERIAL PRIMARY KEY, name TEXT, last_beat TIMESTAMP WITH TIME ZONE, uid TEXT, group_id INT, status TEXT);
CREATE TABLE IF NOT EXISTS "hatchery_model" (hatchery_id BIGINT, worker_model_id BIGINT, PRIMARY KEY(hatchery_id, worker_model_id));

CREATE TABLE IF NOT EXISTS "repositories_manager" (id BIGSERIAL PRIMARY KEY , type TEXT, name TEXT UNIQUE, url TEXT UNIQUE, data JSONB );
CREATE TABLE IF NOT EXISTS "repositories_manager_project" ( id_repositories_manager BIGINT NOT NULL, id_project BIGINT NOT NULL, data JSONB, PRIMARY KEY(id_repositories_manager, id_project));

CREATE TABLE IF NOT EXISTS "stats" (day DATE PRIMARY KEY, build BIGINT, unit_test BIGINT, testing BIGINT, deployment BIGINT, max_building_worker BIGINT, max_building_pipeline BIGINT);
CREATE TABLE IF NOT EXISTS "activity" (day DATE, project_id BIGINT, application_id BIGINT, build BIGINT, unit_test BIGINT, testing BIGINT, deployment BIGINT, PRIMARY KEY(day, project_id, application_id));

CREATE TABLE IF NOT EXISTS "warning" (id BIGSERIAL PRIMARY KEY, project_id BIGINT, app_id BIGINT, pip_id BIGINT, env_id BIGINT, action_id BIGINT, warning_id BIGINT, message_param JSONB);

GRANT SELECT, INSERT, UPDATE, DELETE on ALL TABLES IN SCHEMA public TO "cds";
GRANT ALL ON ALL SEQUENCES IN SCHEMA public TO "cds";
